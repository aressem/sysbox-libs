package pidmonitor

import (
	"fmt"
	"os"
	"time"
)

type cmd int

const (
	stop cmd = iota
)

// Monitors events associated with the given PidMon instance
func pidMonitor(pm *PidMon) {

	for {
		eventList := []PidEvent{}
		rmList := []PidEvent{}

		// handle incoming commands first
		select {
		case cmd := <-pm.cmdCh:
			if cmd == stop {
				pm.EventCh <- eventList
				return
			}
		default:
		}

		// perform monitoring action
		pm.mu.Lock()
		for pid, evect := range pm.eventTable {
			if eventIsSet(evect, Exit) {
				pidAlive, err := pidExists(pid)
				if err != nil || !pidAlive {

					eventList = append(eventList, PidEvent{
						Pid:   pid,
						Event: Exit,
						Err:   err,
					})

					// pid exit implies event won't hit again; remove it.
					rmList = append(rmList, PidEvent{pid, Exit, nil})
				}
			}
		}

		// send event list
		if len(eventList) > 0 {
			pm.EventCh <- eventList
		}

		// remove events that won't hit any more
		for _, e := range rmList {
			eventTableRm(pm.eventTable, e)
		}

		pm.mu.Unlock()

		// wait for the poll period
		time.Sleep(pm.cfg.Poll * time.Millisecond)
	}
}

// Checks if a process with the given pid exists.
func pidExists(pid uint32) (bool, error) {

	// Our current checking mechanism is very simple but not the best; in the future, we
	// should consider replacing it with the newly added pidfd_* syscalls in Linux.

	path := fmt.Sprintf("/proc/%d", pid)

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}
