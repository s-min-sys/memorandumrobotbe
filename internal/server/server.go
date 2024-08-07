package server

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	scheduletask "github.com/patdz/schedule_task"
	"github.com/s-min-sys/memorandumrobotbe/internal/config"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/stg/mwf"
)

type MemoData struct {
	Memos       map[string]*Memo
	MemoRecords map[string]*MemoRecord
}

func (md *MemoData) tryInit() {
	if md.Memos == nil {
		md.Memos = make(map[string]*Memo)
	}

	if md.MemoRecords == nil {
		md.MemoRecords = make(map[string]*MemoRecord)
	}
}

type Server struct {
	wg        sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc
	cfg       *config.Config
	logger    l.Wrapper

	memos    *mwf.MemWithFile[*MemoData, mwf.Serial, mwf.Lock]
	taskPool scheduletask.ScheduleTaskPool
}

func NewServer(cfg *config.Config, logger l.Wrapper) *Server {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	logger = logger.WithFields(l.StringField(l.ClsKey, "Server"))

	if cfg == nil || cfg.Listens == "" {
		logger.Fatal("no valid config")
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		ctx:       ctx,
		ctxCancel: cancel,
		cfg:       cfg,
		logger:    logger,
		memos: mwf.NewMemWithFile[*MemoData, mwf.Serial, mwf.Lock](
			&MemoData{}, &mwf.JSONSerial{
				MarshalIndent: cfg.Debug,
			}, &sync.RWMutex{}, filepath.Join(cfg.Root, "memos.json"), nil),
		taskPool: scheduletask.NewTimeWheelTaskPool(),
	}

	s.init()

	return s
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) init() {
	doNotify(s.logger, s.cfg.NotifierURL, "memorandum robot start")

	s.memos.Read(func(d *MemoData) {
		for id, memo := range d.Memos {
			if memo.Disabled {
				continue
			}

			mr, ok := d.MemoRecords[id]
			if !ok {
				s.logger.WithFields(l.StringField("memID", id)).Error("no memo record")

				continue
			}

			timeNow := time.Now()

			if timeNow.Sub(mr.LastTouchAt) > memo.Span {
				s.notifyMemo(mr.LastTouchAt, memo.ID, memo.Name, memo.Info)

				continue
			}

			err := s.taskPool.AddTask(memo.ID, mr.LastTouchAt.Add(memo.Span), s.taskCb, memo.ID)
			if err != nil {
				s.logger.WithFields(l.ErrorField(err)).Fatal("add task failed!!")
			}
		}
	})

	s.wg.Add(1)

	go s.httpRoutine()

	s.wg.Add(1)

	go s.goodMorningRoutine()
}

func (s *Server) goodMorningRoutine() {
	defer s.wg.Done()

	loop := true

	for loop {
		select {
		case <-s.ctx.Done():
			loop = false

			break
		case <-time.After(time.Hour):
			if time.Now().Hour() == 9 {
				s.reNotify()
			}
		}
	}
}

//
//
//

func (s *Server) taskCb(args ...interface{}) {
	if len(args) < 1 {
		s.logger.Fatal("invalid taskCB params")
	}

	s.notifyMemoWithID(args[0].(string)) // nolint:forcetypeassert
}

//
//
//

//
//
//

func (s *Server) notifyMemoWithID(memID string) {
	s.memos.Read(func(md *MemoData) {
		if md == nil || len(md.Memos) == 0 {
			return
		}

		memo, ok := md.Memos[memID]
		if !ok {
			return
		}

		mr, ok := md.MemoRecords[memID]
		if !ok {
			return
		}

		s.notifyMemo(mr.LastTouchAt, memo.ID, memo.Name, memo.Info)
	})
}

func (s *Server) notifyMemo(lastTouchAt time.Time, memoID, memoName, memoInfo string) {
	doNotify(s.logger, s.cfg.NotifierURL, fmt.Sprintf(`
%s 过期
ID: %s
信息: %s
上次成功触发时间: %s
`, memoName, memoID, memoInfo, lastTouchAt.Format("2006-01-02 15:04:05")))
}
