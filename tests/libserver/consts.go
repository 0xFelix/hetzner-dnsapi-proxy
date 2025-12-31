package libserver

const (
	TLD                   = "tld"
	ZoneName              = "test.tld"
	ARecordName           = "asub"
	ARecordNameFull       = "asub.test.tld"
	TXTRecordNameNoPrefix = "txtsub.test.tld"
	TXTRecordName         = "_acme-challenge.txtsub"
	TXTRecordNameFull     = "_acme-challenge.txtsub.test.tld"
	DefaultTTL            = 60
	AExisting             = "127.0.0.1"
	AUpdated              = "1.2.3.4"
	TXTExisting           = "randomvalue"
	TXTUpdated            = "changedrandomvalue"
	RecordTypeA           = "A"
	RecordTypeTXT         = "TXT"

	ZoneID      = "1"
	ARecordID   = "1"
	TXTRecordID = "2"
)
