package cache

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dkrichards86/gocache/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testKey = "key"
)

var testElem *Item = &Item{
	Value: "hey oh",
}

type slowAdapter struct {
	elem *Item
}

func (me *slowAdapter) Query(key string) (interface{}, error) {
	time.Sleep(time.Microsecond * 2)
	return me.elem, nil
}

func newSlowAdapter(elem *Item) *slowAdapter {
	return &slowAdapter{elem}
}

func doCacheTest(t *testing.T, testCache Cache, concurrentReads int) {
	i, err := testCache.Get(testKey)
	assert.Equal(t, i.Value, testElem.Value)
	require.NoError(t, err)

	var wg sync.WaitGroup
	resultsChan := make(chan *Item)
	errChan := make(chan error)
	successfulGets := 0
	unsuccessfulGets := 0
	errorGets := 0

	go func() {
		for {
			select {
			case i, ok := <-resultsChan:
				if !ok {
					resultsChan = nil
					continue
				}
				if i != nil && reflect.DeepEqual(i.Value, testElem.Value) {
					successfulGets++
				} else {
					unsuccessfulGets++
				}
			case err, ok := <-errChan:
				if !ok {
					errChan = nil
					continue
				}
				if err != nil {
					errorGets++
				}
			}

			if resultsChan == nil && errChan == nil {
				break
			}
		}
	}()

	for i := 0; i < concurrentReads; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			i, err := testCache.Get(testKey)
			resultsChan <- i
			errChan <- err
		}(&wg)
	}

	wg.Wait()
	close(resultsChan)
	close(errChan)

	assert.Equal(t, concurrentReads, successfulGets)
	assert.Equal(t, 0, unsuccessfulGets)
	assert.Equal(t, 0, errorGets)
}

func TestSimpleCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewSimpleCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 0)
}

func TestConcurrentCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewConcurrentCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 0)
}

func TestLockedCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewLockedCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 0)
}

func TestSingleflightCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewCoalescedCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 0)
}

func TestSimpleCache_Concurrent(t *testing.T) {
	t.Skip("this is not thread safe")
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewSimpleCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 10000)
}

func TestConcurrentCache_Concurrent(t *testing.T) {
	t.Skip("this is not thread safe")
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewConcurrentCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 10000)
}

func TestLockedCache_Concurrent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewLockedCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 10000)
}

func TestSingleflightCache_Concurrent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockAdapter := mocks.NewMockAdapter(mockCtrl)
	mockAdapter.EXPECT().Query(testKey).AnyTimes().Return(testElem, nil)
	testCache := NewCoalescedCache(mockAdapter, time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 10000)
}

func TestLockedCache_SlowAdapter(t *testing.T) {
	testCache := NewLockedCache(newSlowAdapter(testElem), time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 10000)
}

func TestCoalescedCache_SlowAdapter(t *testing.T) {
	testCache := NewCoalescedCache(newSlowAdapter(testElem), time.Duration(time.Microsecond))
	doCacheTest(t, testCache, 10000)
}
