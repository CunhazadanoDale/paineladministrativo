package comissao

import (
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type LinhaResumoResponse struct {
	EmpresaID         uuid.UUID `json:"empresa_id"`
	RepresentanteID   uuid.UUID `json:"representante_id"`
	RepresentanteNome string    `json:"representante_nome"`
	ComissoesCentavos int64     `json:"comissoes_centavos"`
	AjustesCentavos   int64     `json:"ajustes_centavos"`
	TotalCentavos     int64     `json:"total_centavos"`
	EmAbertoCentavos  int64     `json:"em_aberto_centavos"`
	FechadoCentavos   int64     `json:"fechado_centavos"`
	PagoCentavos      int64     `json:"pago_centavos"`
}

type ResumoResponse struct {
	Competencia   string                `json:"competencia"`
	TotalCentavos int64                 `json:"total_centavos"`
	Linhas        []LinhaResumoResponse `json:"linhas"`
}

func NovoResumoResponse(resumo domaincomissao.ResumoComissoes) ResumoResponse {
	linhas := make([]LinhaResumoResponse, 0, len(resumo.Linhas))
	for _, linha := range resumo.Linhas {
		linhas = append(linhas, LinhaResumoResponse{
			EmpresaID:         linha.EmpresaID,
			RepresentanteID:   linha.RepresentanteID,
			RepresentanteNome: linha.RepresentanteNome,
			ComissoesCentavos: linha.ComissoesCentavos,
			AjustesCentavos:   linha.AjustesCentavos,
			TotalCentavos:     linha.ComissoesCentavos + linha.AjustesCentavos,
			EmAbertoCentavos:  linha.EmAbertoCentavos,
			FechadoCentavos:   linha.FechadoCentavos,
			PagoCentavos:      linha.PagoCentavos,
		})
	}

	return ResumoResponse{
		Competencia:   resumo.Competencia.String(),
		TotalCentavos: resumo.TotalCentavos(),
		Linhas:        linhas,
	}
}
