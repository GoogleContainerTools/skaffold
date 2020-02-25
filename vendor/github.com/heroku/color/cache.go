package color

import "sync"

var cacheSingleton *colorCache
var cacheOnce sync.Once

func cache() *colorCache {
	cacheOnce.Do(func() {
		cacheSingleton = &colorCache{
			cache: make(colorMap),
		}
	})
	return cacheSingleton
}

type colorMap map[Attribute]*Color

type colorCache struct {
	sync.RWMutex
	cache colorMap
}

func (cc *colorCache) value(attrs ...Attribute) *Color {
	key := to_key(attrs)
	if v := cc.getIfExists(key); v != nil {
		return v
	}
	cc.Lock()
	v := &Color{
		colorStart: chainSGRCodes(attrs),
	}
	cc.cache[key] = v
	cc.Unlock()
	return v
}

func (vc *colorCache) getIfExists(key Attribute) *Color {
	vc.RLock()
	if v, ok := vc.cache[key]; ok {
		vc.RUnlock()
		return v

	}
	vc.RUnlock()
	return nil
}

func (vc *colorCache) clear() {
	vc.Lock()
	defer vc.Unlock()
	vc.cache = make(colorMap)
}
