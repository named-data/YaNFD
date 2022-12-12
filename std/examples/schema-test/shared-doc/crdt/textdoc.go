package crdt

import (
	"github.com/zjkmxy/go-ndn/pkg/utils/priority_queue"
)

type TextDoc struct {
	doc        Doc[string]
	pendingOps priority_queue.Queue[*Record, uint64]
}

func (td *TextDoc) HandleRecord(record *Record) {
	if record.ID == nil || (record.RecordType != RecordInsert && record.RecordType != RecordDelete) {
		return
	}
	td.pendingOps.Push(record, record.ID.Clock)
	for td.pendingOps.Len() > 0 {
		rec := td.pendingOps.Peek()
		success := false
		if rec.RecordType == RecordInsert {
			success = td.doc.Insert(rec.ID, rec.Origin, rec.RightOrigin, rec.Content)
		} else {
			success = td.doc.Delete(rec.ID)
		}
		if !success {
			break
		}
		td.pendingOps.Pop()
	}
}

func (td *TextDoc) GetText() string {
	ret := ""
	for item := td.doc.Start; item != nil; item = item.Right {
		if !item.Deleted {
			ret += item.Content
		}
	}
	return ret
}

func (td *TextDoc) Insert(offset int, content string) *Record {
	item := td.doc.LocalInsert(offset, content)
	if item == nil {
		return nil
	}
	return &Record{
		RecordType:  RecordInsert,
		ID:          &item.ID,
		Origin:      item.Origin,
		RightOrigin: item.RightOrigin,
		Content:     content,
	}
}

func (td *TextDoc) Delete(offset int) *Record {
	item := td.doc.LocalDelete(offset)
	if item == nil {
		return nil
	}
	return &Record{
		RecordType: RecordDelete,
		ID:         &item.ID,
	}
}

func NewTextDoc(producer uint64) *TextDoc {
	return &TextDoc{
		doc: Doc[string]{
			Producer: producer,
			Clock:    0,
			Start:    nil,
		},
		pendingOps: priority_queue.New[*Record, uint64](),
	}
}
