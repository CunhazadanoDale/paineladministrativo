package empresa

import "github.com/google/uuid"

type Vinculos struct {
	AcessoTodas bool
	EmpresaIDs  []uuid.UUID
}
