package gorace

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type GoRaceTestSuite struct {
	suite.Suite
}

func (suite *GoRaceTestSuite) BeforeTest() {
}

func (suite *GoRaceTestSuite) AfterTest() {
}

func (suite *GoRaceTestSuite) TestGoRaceCancelable() {
	cancelable := GoRace(func(ctx context.Context, cancelable GoCancelable) {
		result := work(ctx)
		cancelable.Send(result)
	})
	cancelable.Start(context.Background())
	result := <-cancelable.Receive()
	suite.Equal(true, result)
}

func (suite *GoRaceTestSuite) TestGoRaceBackground() {
	cancelable := GoRace(func(ctx context.Context, cancelable GoCancelable) {
		result := work(ctx)
		cancelable.Send(result)
	})
	cancelable.StartBackground(context.Background())
	result := <-cancelable.Receive()
	suite.Equal(true, result)
}

func (suite *GoRaceTestSuite) TestGoRaceCheckForDeadlocks() {
	cancelable := GoRace(func(ctx context.Context, cancelable GoCancelable) {
		result := work(ctx)
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(wg *sync.WaitGroup, result bool, i int) {
				defer wg.Done()
				cancelable.Send(result)
			}(&wg, result, i)
		}
		wg.Wait()
	})
	cancelable.StartBackground(context.Background())

	for result := range cancelable.Receive() {
		suite.Equal(true, result, "<-cancelable.Receive() failed")
	}

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
}

func (suite *GoRaceTestSuite) TestGoRaceCheckForCancelDeadlocks() {
	cancelable := GoRace(func(ctx context.Context, cancelable GoCancelable) {
		work(ctx)
		for i := 0; i < 50; i++ {
			cancelable.Send(i)
			cancelable.Cancel()
		}
	})
	cancelable.Start(context.Background())

	for result := range cancelable.Receive() {
		suite.Equal(0, result, "cancelable.Receive() should be 0 due to cancel immediately after first send")
	}

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
}

func (suite *GoRaceTestSuite) TestGoRaceCheckForStartBackgroundPanics() {
	cancelable := rapidSendCancelable()
	cancelable.StartBackground(context.Background())

	for result := range cancelable.Receive() {
		suite.Equal(true, result, "<-cancelable.Receive() failed")
	}

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
}

func (suite *GoRaceTestSuite) TestGoRaceCheckForStartPanics() {
	cancelable := rapidSendCancelable()

	for result := range cancelable.Start(context.Background()).Receive() {
		suite.Equal(true, result, "<-cancelable.Receive() failed")
	}

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
}

func (suite *GoRaceTestSuite) TestGoRaceCheckCancel() {
	cancelable := rapidSendCancelable()

	cancelable.Cancel()

	for result := range cancelable.Start(context.Background()).Receive() {
		suite.Equal(true, result, "<-cancelable.Receive() failed")
	}

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
}

func (suite *GoRaceTestSuite) TestGoRaceCheckCancelForStartBackground() {
	cancelable := rapidSendCancelable()

	cancelable.StartBackground(context.Background())
	cancelable.Cancel()

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
	suite.Nil(cancelable.LastResult(), "cancelable.LastResult() should be nil")
}

func (suite *GoRaceTestSuite) TestGoRaceCheckMultipleCancels() {
	cancelable := rapidSendCancelable()

	cancelable.StartBackground(context.Background())

	for i := 0; i < 50; i++ {
		cancelable.Cancel()
	}

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
	suite.Nil(cancelable.LastResult(), "cancelable.LastResult() should be nil")
}

func (suite *GoRaceTestSuite) TestGoRaceCheckMultipleStarts() {
	cancelable := rapidSendCancelable()

	for i := 0; i < 50; i++ {
		cancelable.Start(context.Background())
		cancelable.StartBackground(context.Background())
	}

	for result := range cancelable.Receive() {
		suite.Equal(true, result, "<-cancelable.Receive() failed")
	}

	suite.Equal(true, cancelable.IsCanceled(), "cancelable.IsCanceled() should be true")
}

func TestGoRaceSuite(t *testing.T) {
	suite.Run(t, new(GoRaceTestSuite))
}

func work(ctx context.Context) bool {
	<-time.After(250 * time.Millisecond)
	return true
}

func rapidSendCancelable() GoCancelable {
	return GoRace(func(ctx context.Context, cancelable GoCancelable) {
		result := work(ctx)
		for i := 0; i < 50; i++ {
			go func(result bool, i int) {
				cancelable.Send(result)
			}(result, i)
		}
	})
}
