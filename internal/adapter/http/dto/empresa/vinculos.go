package empresa

import (
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/empresa"
	"github.com/google/uuid"
)

type VinculosRequest struct {
	AcessoTodas bool        `json:"acesso_todas"`
	EmpresaIDs  []uuid.UUID `json:"empresa_ids"`
}

func (r VinculosRequest) ParaInput(usuarioID, alvoID uuid.UUID) portsin.DefinirVinculosInput {
	return portsin.DefinirVinculosInput{
		UsuarioID:   usuarioID,
		AlvoID:      alvoID,
		AcessoTodas: r.AcessoTodas,
		EmpresaIDs:  r.EmpresaIDs,
	}
}

type VinculosResponse struct {
	AcessoTodas bool        `json:"acesso_todas"`
	EmpresaIDs  []uuid.UUID `json:"empresa_ids"`
}

func NovaVinculosResponse(vinculos *domainempresa.Vinculos) VinculosResponse {
	empresaIDs := make([]uuid.UUID, 0, len(vinculos.EmpresaIDs))
	empresaIDs = append(empresaIDs, vinculos.EmpresaIDs...)

	return VinculosResponse{
		AcessoTodas: vinculos.AcessoTodas,
		EmpresaIDs:  empresaIDs,
	}
}
