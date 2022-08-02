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
