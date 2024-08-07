package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sgostarter/i/commerr"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/ptl"
)

func (s *Server) httpRoutine() {
	defer s.wg.Done()

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(gin.Recovery())
	r.Use(requestid.New())

	r.Any("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello")
	})

	subGroup := r.Group("/api")

	subGroup.POST("/add", s.handleAdd)
	subGroup.POST("/del", s.handleDel)
	subGroup.POST("/touch", s.handleTouch)
	subGroup.GET("/simple-touch/:id", s.handleSimpleTouch)
	subGroup.POST("/renotify", s.handleReNotify)
	subGroup.POST("/all", s.handleAll)

	listenAddresses := s.cfg.Listens

	fnListen := func(listen string) {
		srv := &http.Server{
			Addr:        listen,
			ReadTimeout: time.Second,
			Handler:     r,
		}

		s.logger.Infof("http server start: %s\n", listen)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Panicln(err)
		}

		go func(cs *http.Server) {
			<-s.ctx.Done()

			_ = cs.Close()
		}(srv)
	}

	listens := strings.Split(listenAddresses, " ")

	for idx := 0; idx < len(listens)-1; idx++ {
		go fnListen(listens[idx])
	}

	fnListen(listens[len(listens)-1])
	fnListen(listens[len(listens)-1])
}

//
//
//

func (s *Server) handleAdd(c *gin.Context) {
	id, code, msg := s.handleAddInner(c)

	var resp ptl.ResponseWrapper

	if resp.Apply(code, msg) {
		resp.Resp = id
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleAddInner(c *gin.Context) (id string, code ptl.Code, msg string) {
	var req AddRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		code = ptl.CodeErrCommunication
		msg = err.Error()

		return
	}

	if !req.Valid() {
		code = ptl.CodeErrInvalidArgs
		msg = "invalid request for add"

		return
	}

	err = s.memos.Change(func(md *MemoData) (newMd *MemoData, err error) {
		id = uuid.NewString()

		newMd = md
		if newMd == nil {
			newMd = &MemoData{}
		}

		newMd.tryInit()

		newMd.Memos[id] = &Memo{
			ID:   id,
			Name: req.Name,
			Info: req.Info,
			Span: time.Duration(req.InternalSeconds) * time.Second,
		}

		newMd.MemoRecords[id] = &MemoRecord{
			ID:          id,
			LastTouchAt: time.Now(),
		}

		return
	})

	if err != nil {
		code = ptl.CodeErrInternal
		msg = err.Error()

		return
	}

	code = ptl.CodeSuccess

	doNotify(s.logger, s.cfg.NotifierURL, fmt.Sprintf("add %s, id is %s", req.Name, id))

	return
}

func (s *Server) handleDel(c *gin.Context) {
	var resp ptl.ResponseWrapper

	resp.Apply(s.handleDelInner(c))

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleDelInner(c *gin.Context) (code ptl.Code, msg string) {
	var req DelRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		code = ptl.CodeErrCommunication
		msg = err.Error()

		return
	}

	if !req.Valid() {
		code = ptl.CodeErrInvalidArgs
		msg = "invalid request"

		return
	}

	_ = s.taskPool.RemoveTask(req.ID)

	_ = s.memos.Change(func(md *MemoData) (newMd *MemoData, err error) {
		newMd = md
		if newMd == nil {
			newMd = &MemoData{}
		}

		newMd.tryInit()

		delete(newMd.Memos, req.ID)
		delete(newMd.MemoRecords, req.ID)

		return
	})

	code = ptl.CodeSuccess

	doNotify(s.logger, s.cfg.NotifierURL, fmt.Sprintf("del %s", req.ID))

	return
}

func (s *Server) handleTouch(c *gin.Context) {
	var resp ptl.ResponseWrapper

	resp.Apply(s.handleTouchInner(c))

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleTouchInner(c *gin.Context) (code ptl.Code, msg string) {
	var req TouchRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		code = ptl.CodeErrCommunication
		msg = err.Error()

		return
	}

	if !req.Valid() {
		code = ptl.CodeErrInvalidArgs
		msg = "invalid request"

		return
	}

	return s.handleTouchInnerWithRequest(&req)
}

func (s *Server) handleTouchInnerWithRequest(req *TouchRequest) (code ptl.Code, msg string) {
	code = ptl.CodeSuccess

	_ = s.memos.Change(func(md *MemoData) (newMd *MemoData, err error) {
		newMd = md
		if newMd == nil {
			newMd = &MemoData{}
		}

		newMd.tryInit()

		memo, ok := newMd.Memos[req.ID]
		if !ok {
			code = ptl.CodeErrNotExists
			msg = fmt.Sprintf("no memo %s", req.ID)

			err = commerr.ErrAborted

			return
		}

		memoRecord, ok := newMd.MemoRecords[req.ID]
		if !ok {
			code = ptl.CodeErrNotExists
			msg = fmt.Sprintf("no memo record %s", req.ID)

			err = commerr.ErrAborted

			return
		}

		timeNow := time.Now()

		memTouchInfo := &MemoTouchInfo{
			At:   timeNow,
			Info: req.Info,
		}

		if req.FailFlag {
			memoRecord.LastFailTouchInfo = memTouchInfo
		} else {
			memoRecord.LastTouchAt = timeNow
			memoRecord.LastSuccessTouchInfo = memTouchInfo

			_ = s.taskPool.RemoveTask(req.ID)

			if !memo.Disabled {
				e := s.taskPool.AddTask(memo.ID, timeNow.Add(memo.Span), s.taskCb, memo.ID)
				if e != nil {
					s.logger.WithFields(l.ErrorField(e), l.StringField("id", memo.ID)).Error("add task failed!!")
				}
			}
		}

		return
	})

	return
}

func (s *Server) handleSimpleTouch(c *gin.Context) {
	var resp ptl.ResponseWrapper

	resp.Apply(s.handleSimpleTouchInner(c))

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleSimpleTouchInner(c *gin.Context) (code ptl.Code, msg string) {
	id := c.Param("id")
	if id == "" {
		code = ptl.CodeErrInvalidArgs
		msg = "no id"

		return
	}

	return s.handleTouchInnerWithRequest(&TouchRequest{
		ID: id,
	})
}

func (s *Server) handleReNotify(c *gin.Context) {
	var resp ptl.ResponseWrapper

	resp.Apply(s.handleReNotifyInner(c))

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleReNotifyInner(_ *gin.Context) (code ptl.Code, msg string) {
	s.reNotify()

	code = ptl.CodeSuccess

	return
}

func (s *Server) reNotify() {
	s.memos.Read(func(md *MemoData) {
		if md == nil || len(md.Memos) == 0 {
			return
		}

		for id, memo := range md.Memos {
			memoRecord, ok := md.MemoRecords[id]
			if !ok {
				continue
			}

			if time.Since(memoRecord.LastTouchAt) >= memo.Span {
				s.notifyMemo(memoRecord.LastTouchAt, memo.ID, memo.Name, memo.Info)
			}
		}
	})
}

func (s *Server) handleAll(c *gin.Context) {
	memoAll, code, msg := s.handleAllInner(c)

	var resp ptl.ResponseWrapper

	if resp.Apply(code, msg) {
		resp.Resp = memoAll
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleAllInner(_ *gin.Context) (memoAll []*MemoAllItem, code ptl.Code, msg string) {
	memoAll = make([]*MemoAllItem, 0, 10)

	s.memos.Read(func(md *MemoData) {
		if md == nil || len(md.Memos) == 0 {
			return
		}

		timeNow := time.Now()

		for id, memo := range md.Memos {
			mr, ok := md.MemoRecords[id]
			if ok {
				var untilExpired string

				var expired bool

				if timeNow.Sub(mr.LastTouchAt) > memo.Span {
					expired = true
				} else {
					untilExpired = (memo.Span - timeNow.Sub(mr.LastTouchAt)).String()
				}

				memoAll = append(memoAll, &MemoAllItem{
					ID:                   memo.ID,
					Name:                 memo.Name,
					Info:                 memo.Info,
					Span:                 memo.Span.String(),
					Disabled:             memo.Disabled,
					LastTouchAt:          mr.LastTouchAt,
					LastSuccessTouchInfo: mr.LastSuccessTouchInfo,
					LastFailTouchInfo:    mr.LastFailTouchInfo,
					UntilExpired:         untilExpired,
					Expired:              expired,
				})
			}
		}
	})

	code = ptl.CodeSuccess

	return
}
