package processor3

import (
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type pythVaaDb struct {
	m map[string]*pythVaaEntry
	l sync.RWMutex
}

type pythVaaEntry struct {
	v          *vaa.VAA
	updateTime time.Time // Used for determining when to delete entries
}

func NewPythVaaDb() pythVaaDb {
	return pythVaaDb{
		m: make(map[string]*pythVaaEntry),
	}
}

func (pdb *pythVaaDb) put(id db.VAAID, v *vaa.VAA) {
	key := fmt.Sprintf("%v/%v", id.EmitterAddress, id.Sequence)
	value := &pythVaaEntry{v: v, updateTime: time.Now()}

	pdb.l.Lock()
	defer pdb.l.Unlock()
	pdb.m[key] = value
}

func (pdb *pythVaaDb) get(id db.VAAID) (*pythVaaEntry, bool) {
	key := fmt.Sprintf("%v/%v", id.EmitterAddress, id.Sequence)

	pdb.l.RLock()
	defer pdb.l.Unlock()
	v, ok := pdb.m[key]
	return v, ok
}

func (pdb *pythVaaDb) deleteBefore(t time.Time) {
	pdb.l.Lock()
	defer pdb.l.Unlock()

	for k, v := range pdb.m {
		if v.updateTime.Before(t) {
			delete(pdb.m, k)
		}
	}
}
