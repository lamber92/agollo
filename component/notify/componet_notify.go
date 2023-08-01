/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package notify

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apolloconfig/agollo/v4/component/log"
	"github.com/apolloconfig/agollo/v4/component/remote"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
)

const (
	longPollInterval = 2 * time.Second // 2s
)

var (
	changeMonitor ChangeMonitor
	initOnce      = sync.Once{}
)

type ChangeMonitor interface {
	Start()
	Stop()
}

// internalMonitor long poling monitor
// regularly and asynchronously monitor whether the remote configuration has changed
type internalMonitor struct {
	appConfigFunc func() config.AppConfig
	cache         *storage.Cache
	running       atomic.Bool

	cancelFunc context.CancelFunc
	cancelLock sync.Mutex

	exitFlag sync.WaitGroup
}

func NewChangeMonitor(appConfigFunc func() config.AppConfig, cache *storage.Cache) ChangeMonitor {
	initOnce.Do(func() {
		changeMonitor = &internalMonitor{
			appConfigFunc: appConfigFunc,
			cache:         cache,
		}
	})
	return changeMonitor
}

// Start start long polling
func (c *internalMonitor) Start() {
	// check if long polling has been started
	if c.running.Load() {
		log.Warn("long polling is running now!")
		return
	}
	c.running.Store(true)
	c.exitFlag.Add(1)
	defer func() {
		c.running.Store(false)
		c.exitFlag.Done()
	}()

	log.Info("long polling started!")

	t2 := time.NewTimer(longPollInterval)
	instance := remote.CreateAsyncApolloConfig()
	// long poll for sync
loop:
	for {
		if !c.cancelLock.TryLock() {
			return
		}
		ctx, cancelFunc := context.WithCancel(context.Background())
		c.cancelFunc = cancelFunc
		c.cancelLock.Unlock()

		select {
		case <-t2.C:
			// block long polling
			configs := instance.Sync(ctx, c.appConfigFunc)
			for _, apolloConfig := range configs {
				c.cache.UpdateApolloConfig(apolloConfig, c.appConfigFunc)
			}
			if cancelFunc != nil {
				cancelFunc()
			}
			if !c.running.Load() {
				break loop
			}

			t2.Reset(longPollInterval)
		}
	}

	log.Info("long polling exist!")
}

// Stop exit long polling
func (c *internalMonitor) Stop() {
	c.cancelLock.Lock()
	defer c.cancelLock.Unlock()

	c.running.Store(false)
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	c.exitFlag.Wait()
}
