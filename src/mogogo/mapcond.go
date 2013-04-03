package mogogo

import (
	"sort"
	"sync"
	"time"
)

type keyset [8]string
type valarray [8]interface{}
type waitlist map[uint]chan bool
type mapCond struct {
	nextId       uint
	Timeout      time.Duration
	l            sync.Locker
	keySets      map[keyset]bool
	waitLists    map[keyset]map[valarray]waitlist
	idToWaitList map[uint]waitlist
}

func newMapCond() *mapCond {
	return &mapCond{
		Timeout:      30 * time.Second,
		l:            &sync.Mutex{},
		keySets:      make(map[keyset]bool),
		waitLists:    make(map[keyset]map[valarray]waitlist),
		idToWaitList: make(map[uint]waitlist),
	}
}
func (mc *mapCond) getKeySet(m map[string]interface{}) keyset {
	var ret keyset
	if len(m) > 8 {
		panic("map len cannot great than 8")
	}
	var s []string = make([]string, 0, len(m))
	for k, _ := range m {
		if k == "" {
			panic("map key cannot empty")
		}
		s = append(s, k)
	}
	sort.Strings(s)
	for i, k := range s {
		ret[i] = k
	}
	return ret
}
func (mc *mapCond) getValArray(m map[string]interface{}, ks keyset) valarray {
	var ret valarray
	for i, k := range ks {
		if k == "" {
			break
		}
		ret[i] = m[k]
	}
	return ret
}
func (mc *mapCond) matchKeySet(ks keyset, m map[string]interface{}) bool {
	for _, k := range ks {
		if k == "" {
			return true
		}
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}
func (mc *mapCond) addWaitList(ks keyset, va valarray) (id uint, wait <-chan bool) {
	mc.keySets[ks] = true
	wls, ok := mc.waitLists[ks]
	var wl waitlist
	if !ok {
		wls = make(map[valarray]waitlist)
		mc.waitLists[ks] = wls
		wl = make(waitlist)
		wls[va] = wl
	} else {
		wl, ok = wls[va]
		if !ok {
			wl = make(waitlist)
			wls[va] = wl
		}
	}
	id = mc.nextId
	mc.nextId++
	wl[id] = make(chan bool, 1)
	wait = wl[id]
	mc.idToWaitList[id] = wl
	return
}
func (mc *mapCond) waitOn(cond map[string]interface{}) (id uint, wait <-chan bool) {
	mc.l.Lock()
	defer mc.l.Unlock()
	ks := mc.getKeySet(cond)
	va := mc.getValArray(cond, ks)
	return mc.addWaitList(ks, va)

}
func (mc *mapCond) removeId(id uint) {
	mc.l.Lock()
	defer mc.l.Unlock()
	wl := mc.idToWaitList[id]
	delete(wl, id)
	delete(mc.idToWaitList, id)
}
func (mc *mapCond) Wait(cond map[string]interface{}) (timeout bool) {
	id, w := mc.waitOn(cond)
	defer mc.removeId(id)
	select {
	case _ = <-w:
		timeout = false
	case _ = <-time.After(mc.Timeout):
		timeout = true
	}
	return
}
func (mc *mapCond) broadcast(ks keyset, va valarray) {
	wl := mc.waitLists[ks][va]
	for _, w := range wl {
		select {
		case w <- true:
		default:
		}
	}
}
func (mc *mapCond) Broadcast(m map[string]interface{}) {
	mc.l.Lock()
	defer mc.l.Unlock()
	for ks, _ := range mc.keySets {
		if mc.matchKeySet(ks, m) {
			va := mc.getValArray(m, ks)
			mc.broadcast(ks, va)
		}
	}
}
