package mogogo

import (
	"reflect"
	"sort"
	"sync"
	"time"
	"fmt"
)

type keyset interface{}

func ks2slice(ks keyset) (ret []string) {
	v := reflect.ValueOf(ks)
	l := v.Len()
	ret = make([]string, 0, v.Len())
	for i := 0; i < l; i++ {
		ret = append(ret, v.Index(i).Interface().(string))
	}
	return
}

type valarray interface{}
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
	var s []string
	switch len(m) {
	case 0:
		a0 := [0]string{}
		ret = a0
		s = a0[:]
	case 1:
		a1 := [1]string{}
		ret = a1
		s = a1[:]
	case 2:
		a2 := [2]string{}
		ret = a2
		s = a2[:]
	case 3:
		a3 := [3]string{}
		ret = a3
		s = a3[:]
	case 4:
		a4 := [4]string{}
		ret = a4
		s = a4[:]
	case 5:
		a5 := [5]string{}
		ret = a5
		s = a5[:]
	case 6:
		a6 := [6]string{}
		ret = a6
		s = a6[:]
	case 7:
		a7 := [7]string{}
		ret = a7
		s = a7[:]
	default:
		panic("max length is 8")
	}
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Strings(s)
	fmt.Println("getks:", ret)
	return ret
}
func (mc *mapCond) getValArray(m map[string]interface{}, ks keyset) valarray {
	var ret valarray
	var s []interface{}
	switch len(m) {
	case 0:
		a0 := [0]interface{}{}
		ret = a0
		s = a0[:]
	case 1:
		a1 := [1]interface{}{}
		ret = a1
		s = a1[:]
	case 2:
		a2 := [2]interface{}{}
		ret = a2
		s = a2[:]
	case 3:
		a3 := [3]interface{}{}
		ret = a3
		s = a3[:]
	case 4:
		a4 := [4]interface{}{}
		ret = a4
		s = a4[:]
	case 5:
		a5 := [5]interface{}{}
		ret = a5
		s = a5[:]
	case 6:
		a6 := [6]interface{}{}
		ret = a6
		s = a6[:]
	case 7:
		a7 := [7]interface{}{}
		ret = a7
		s = a7[:]
	default:
		panic("max length is 8")
	}
	for i, k := range ks2slice(ks) {
		s[i] = m[k]
	}
	return ret
}
func (mc *mapCond) matchKeySet(ks keyset, m map[string]interface{}) bool {
	for _, k := range ks2slice(ks) {
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
		wl, ok := wls[va]
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
	fmt.Println("wt:", ks, va)
	return mc.addWaitList(ks, va)

}
func (mc *mapCond) removeId(id uint) {
	mc.l.Lock()
	defer mc.l.Unlock()
	wl := mc.idToWaitList[id]
	delete(wl, id)
}
func (mc *mapCond) Wait(cond map[string]interface{}) (timeout bool) {
	id, w := mc.waitOn(cond)
	select {
	case _ = <-w:
		timeout = false
	case _ = <-time.After(mc.Timeout):
		timeout = true
	}
	mc.removeId(id)
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
		fmt.Println("bc:", ks)
		if mc.matchKeySet(ks, m) {
			va := mc.getValArray(m, ks)
			fmt.Println("bc:", ks, va)
			mc.broadcast(ks, va)
		}
	}
}
