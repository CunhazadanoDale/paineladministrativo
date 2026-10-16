package receber

import (
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type CriarAntecipacaoRequest struct {
	EmpresaID            uuid.UUID   `json:"empresa_id"`
	FornecedorID         uuid.UUID   `json:"fornecedor_id"`
	Numero               *string     `json:"numero"`
	OperadoEm            string      `json:"operado_em"`
	TaxaBP               int         `json:"taxa_bp"`
	ValorLiquidoCentavos int64       `json:"valor_liquido_centavos"`
	ContaBancariaID      uuid.UUID   `json:"conta_bancaria_id"`
	TituloIDs            []uuid.UUID `json:"titulo_ids"`
}

type EstornarAntecipacaoRequest struct {
	Versao int    `json:"versao"`
	Motivo string `json:"motivo"`
}

type RecompraRequest struct {
	Versao                int       `json:"versao"`
	RecompradoEm          string    `json:"recomprado_em"`
	EncargosCentavos      int64     `json:"encargos_centavos"`
	Vencimento            string    `json:"vencimento"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	Motivo                string    `json:"motivo"`
}

type RepasseRequest struct {
	Vencimento            string    `json:"vencimento"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
}

type AntecipacaoTituloResponse struct {
	TituloID       uuid.UUID `json:"titulo_id"`
	ValorCentavos  int64     `json:"valor_centavos"`
	StatusAnterior string    `json:"status_anterior"`
	Vigente        bool      `json:"vigente"`
}

type AntecipacaoResponse struct {
	ID                   uuid.UUID                   `json:"id"`
	EmpresaID            uuid.UUID                   `json:"empresa_id"`
	FornecedorID         uuid.UUID                   `json:"fornecedor_id"`
	Numero               *string                     `json:"numero"`
	OperadoEm            string                      `json:"operado_em"`
	TaxaBP               int                         `json:"taxa_bp"`
	ValorBrutoCentavos   int64                       `json:"valor_bruto_centavos"`
	ValorLiquidoCentavos int64                       `json:"valor_liquido_centavos"`
	EncargosCentavos     int64                       `json:"encargos_centavos"`
	ContaBancariaID      uuid.UUID                   `json:"conta_bancaria_id"`
	Status               string                      `json:"status"`
	EstornoMotivo        *string                     `json:"estorno_motivo"`
	EstornadoEm          *time.Time                  `json:"estornado_em"`
	UsuarioID            uuid.UUID                   `json:"usuario_id"`
	Versao               int                         `json:"versao"`
	CriadoEm             time.Time                   `json:"criado_em"`
	AtualizadoEm         time.Time                   `json:"atualizado_em"`
	Titulos              []AntecipacaoTituloResponse `json:"titulos"`
}

type RecompraResponse struct {
	ID                    uuid.UUID `json:"id"`
	TituloID              uuid.UUID `json:"titulo_id"`
	AntecipacaoID         uuid.UUID `json:"antecipacao_id"`
	RecompradoEm          string    `json:"recomprado_em"`
	EncargosCentavos      int64     `json:"encargos_centavos"`
	ValorCentavos         int64     `json:"valor_centavos"`
	Vencimento            string    `json:"vencimento"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	SolicitacaoID         uuid.UUID `json:"solicitacao_id"`
	Motivo                string    `json:"motivo"`
	UsuarioID             uuid.UUID `json:"usuario_id"`
	CriadoEm              time.Time `json:"criado_em"`
}

type RecompraRegistradaResponse struct {
	Titulo   TituloResponse   `json:"titulo"`
	Recompra RecompraResponse `json:"recompra"`
}

type RepasseResponse struct {
	ID                    uuid.UUID `json:"id"`
	AntecipacaoID         uuid.UUID `json:"antecipacao_id"`
	ValorCentavos         int64     `json:"valor_centavos"`
	Vencimento            string    `json:"vencimento"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	SolicitacaoID         uuid.UUID `json:"solicitacao_id"`
}

func NovaAntecipacaoResponse(item *domainreceber.Antecipacao) AntecipacaoResponse {
	resposta := AntecipacaoResponse{
		ID:                   item.ID,
		EmpresaID:            item.EmpresaID,
		FornecedorID:         item.FornecedorID,
		Numero:               item.Numero,
		OperadoEm:            formatarData(item.OperadoEm),
		TaxaBP:               item.TaxaBP,
		ValorBrutoCentavos:   item.ValorBrutoCentavos,
		ValorLiquidoCentavos: item.ValorLiquidoCentavos,
		EncargosCentavos:     item.EncargosCentavos(),
		ContaBancariaID:      item.ContaBancariaID,
		Status:               item.Status.String(),
		EstornoMotivo:        item.EstornoMotivo,
		EstornadoEm:          item.EstornadoEm,
		UsuarioID:            item.UsuarioID,
		Versao:               item.Versao,
		CriadoEm:             item.CriadoEm,
		AtualizadoEm:         item.AtualizadoEm,
		Titulos:              make([]AntecipacaoTituloResponse, 0, len(item.Titulos)),
	}
	for _, vinculo := range item.Titulos {
		resposta.Titulos = append(resposta.Titulos, AntecipacaoTituloResponse{
			TituloID:       vinculo.TituloID,
			ValorCentavos:  vinculo.ValorCentavos,
			StatusAnterior: vinculo.StatusAnterior.String(),
			Vigente:        vinculo.Vigente,
		})
	}

	return resposta
}

func NovaAntecipacaoResponses(itens []*domainreceber.Antecipacao) []AntecipacaoResponse {
	respostas := make([]AntecipacaoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaAntecipacaoResponse(item))
	}

	return respostas
}

func NovaRecompraRegistradaResponse(titulo *domainreceber.Titulo, recompra *domainreceber.Recompra) RecompraRegistradaResponse {
	return RecompraRegistradaResponse{
		Titulo: NovoTituloResponse(titulo),
		Recompra: RecompraResponse{
			ID:                    recompra.ID,
			TituloID:              recompra.TituloID,
			AntecipacaoID:         recompra.AntecipacaoID,
			RecompradoEm:          formatarData(recompra.RecompradoEm),
			EncargosCentavos:      recompra.EncargosCentavos,
			ValorCentavos:         recompra.ValorCentavos,
			Vencimento:            formatarData(recompra.Vencimento),
			CategoriaFinanceiraID: recompra.CategoriaFinanceiraID,
			SolicitacaoID:         recompra.SolicitacaoID,
			Motivo:                recompra.Motivo,
			UsuarioID:             recompra.UsuarioID,
			CriadoEm:              recompra.CriadoEm,
		},
	}
}

func novoRepasseResponse(repasse *domainreceber.Repasse) *RepasseResponse {
	if repasse == nil {
		return nil
	}

	return &RepasseResponse{
		ID:                    repasse.ID,
		AntecipacaoID:         repasse.AntecipacaoID,
		ValorCentavos:         repasse.ValorCentavos,
		Vencimento:            formatarData(repasse.Vencimento),
		CategoriaFinanceiraID: repasse.CategoriaFinanceiraID,
		SolicitacaoID:         repasse.SolicitacaoID,
	}
}
