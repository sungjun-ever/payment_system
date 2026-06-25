package pg

type PGResponse int

// 결제 요청 오류를 완료, 거절, 실패원인 서버, 실패원인 PG, 미식별
const (
	Succeeded PGResponse = iota
	Completed
	Rejected
	ServerFailed
	PGFailed
	Unknown
)
