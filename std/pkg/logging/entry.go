// Abandoned code. For remark only. Please use apex/log instead.
package log

import (
	"encoding/json"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

const TimeFormat = "2006-01-02 15:04:05.000 -07:00"

type Entry struct {
	Time  time.Time
	Level string
	Mod   string
	Msg   string
	Name  enc.Name
}

type jsonEntry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Mod   string `json:"module"`
	Msg   string `json:"message"`
	Name  string `json:"name,omitempty"`
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	je := &jsonEntry{
		Time:  e.Time.Format(TimeFormat),
		Level: e.Level,
		Mod:   e.Mod,
		Msg:   e.Msg,
		Name:  "",
	}
	if e.Name != nil {
		je.Name = e.Name.String()
	}
	return json.Marshal(je)
}

func (e *Entry) UnmarshalJSON(text []byte) error {
	je := &jsonEntry{}
	err := json.Unmarshal([]byte(text), je)
	if err != nil {
		return err
	}
	e.Time, err = time.Parse(TimeFormat, je.Time)
	if err != nil {
		return err
	}
	if je.Name != "" {
		e.Name, err = enc.NameFromStr(je.Name)
		if err != nil {
			return err
		}
	} else {
		e.Name = nil
	}
	e.Level = je.Level
	e.Mod = je.Mod
	e.Msg = je.Msg
	return nil
}

/*
Explanation on logging level:

	DEBUG: detailed informatoin with raw bytes
		* Packet raw bytes

	INFO: general information on traffic, with name only
		* Packet received
		* Interest expressed

	WARN: some failure that one consider it may happen actually occured
		* No route
		* Fragmentation not supported
		* Packet dropped
		* Validation failure
		* Received Nack or Data for unknown Interests. (Most of the case should be Interest timed out)

	ERROR: some failure that one does not expect to happen actually occured
		* Unable to parse or wrong packet type
		* Face disconnection

	FATAL: some failure that is unable to recover happened.
		* Unreachable code branch reached

Explanation on apex/log fields:

	module: the code module that generates the log entry
	name: the packate name associated to the log entry. Maybe empty.
*/
