package importacao

import (
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

type AlocacaoAbertaResponse struct {
	ImportacaoID     uuid.UUID `json:"importacao_id"`
	ImportacaoNumero int       `json:"importacao_numero"`
	ImportacaoStatus string    `json:"importacao_status"`
	EmpresaID        uuid.UUID `json:"empresa_id"`
	ChegadaPrevista  *string   `json:"chegada_prevista"`
	PedidoID         uuid.UUID `json:"pedido_id"`
	PedidoNumero     int       `json:"pedido_numero"`
	PedidoItemID     uuid.UUID `json:"pedido_item_id"`
	ProdutoID        uuid.UUID `json:"produto_id"`
	Quantidade       int       `json:"quantidade"`
	PrazoLimite      *string   `json:"prazo_limite"`
}

type PrazoEmRiscoResponse struct {
	AlocacaoAbertaResponse
	ChegadaComFolga string `json:"chegada_com_folga"`
	DiasDeAtraso    int    `json:"dias_de_atraso"`
}

type PedidoSemImportacaoResponse struct {
	PedidoID     uuid.UUID `json:"pedido_id"`
	PedidoNumero int       `json:"pedido_numero"`
	EmpresaID    uuid.UUID `json:"empresa_id"`
	PedidoItemID uuid.UUID `json:"pedido_item_id"`
	ProdutoID    uuid.UUID `json:"produto_id"`
	PrazoLimite  *string   `json:"prazo_limite"`
	Restante     int       `json:"restante"`
	Reservado    int       `json:"reservado"`
	Alocado      int       `json:"alocado"`
	SemCobertura int       `json:"sem_cobertura"`
}

type AlertasResponse struct {
	FolgaDias     int                           `json:"folga_dias"`
	PrazoEmRisco  []PrazoEmRiscoResponse        `json:"prazo_em_risco"`
	SemPrevisao   []AlocacaoAbertaResponse      `json:"sem_previsao"`
	SemImportacao []PedidoSemImportacaoResponse `json:"sem_importacao"`
}

type AtendimentoResponse struct {
	ImportacaoID     uuid.UUID `json:"importacao_id"`
	ImportacaoNumero int       `json:"importacao_numero"`
	ImportacaoStatus string    `json:"importacao_status"`
	ChegadaPrevista  *string   `json:"chegada_prevista"`
	RecebidaEm       *string   `json:"recebida_em"`
	PedidoItemID     uuid.UUID `json:"pedido_item_id"`
	ProdutoID        uuid.UUID `json:"produto_id"`
	Quantidade       int       `json:"quantidade"`
	Reservada        *int      `json:"reservada"`
}

func novaAlocacaoAbertaResponse(alocacao domainimportacao.AlocacaoAberta) AlocacaoAbertaResponse {
	return AlocacaoAbertaResponse{
		ImportacaoID:     alocacao.ImportacaoID,
		ImportacaoNumero: alocacao.ImportacaoNumero,
		ImportacaoStatus: alocacao.ImportacaoStatus.String(),
		EmpresaID:        alocacao.EmpresaID,
		ChegadaPrevista:  formatarDataOpcional(alocacao.ChegadaPrevista),
		PedidoID:         alocacao.PedidoID,
		PedidoNumero:     alocacao.PedidoNumero,
		PedidoItemID:     alocacao.PedidoItemID,
		ProdutoID:        alocacao.ProdutoID,
		Quantidade:       alocacao.Quantidade,
		PrazoLimite:      formatarDataOpcional(alocacao.PrazoLimite),
	}
}

func NovosAlertasResponse(alertas *domainimportacao.Alertas) AlertasResponse {
	resposta := AlertasResponse{
		FolgaDias:     alertas.Folga,
		PrazoEmRisco:  make([]PrazoEmRiscoResponse, 0, len(alertas.PrazoEmRisco)),
		SemPrevisao:   make([]AlocacaoAbertaResponse, 0, len(alertas.SemPrevisao)),
		SemImportacao: make([]PedidoSemImportacaoResponse, 0, len(alertas.SemImportacao)),
	}
	for _, alerta := range alertas.PrazoEmRisco {
		resposta.PrazoEmRisco = append(resposta.PrazoEmRisco, PrazoEmRiscoResponse{
			AlocacaoAbertaResponse: novaAlocacaoAbertaResponse(alerta.AlocacaoAberta),
			ChegadaComFolga:        alerta.ChegadaComFolga.Format(formatoData),
			DiasDeAtraso:           alerta.DiasDeAtraso,
		})
	}
	for _, alocacao := range alertas.SemPrevisao {
		resposta.SemPrevisao = append(resposta.SemPrevisao, novaAlocacaoAbertaResponse(alocacao))
	}
	for _, pedido := range alertas.SemImportacao {
		resposta.SemImportacao = append(resposta.SemImportacao, PedidoSemImportacaoResponse{
			PedidoID:     pedido.PedidoID,
			PedidoNumero: pedido.PedidoNumero,
			EmpresaID:    pedido.EmpresaID,
			PedidoItemID: pedido.PedidoItemID,
			ProdutoID:    pedido.ProdutoID,
			PrazoLimite:  formatarDataOpcional(pedido.PrazoLimite),
			Restante:     pedido.Restante,
			Reservado:    pedido.Reservado,
			Alocado:      pedido.Alocado,
			SemCobertura: pedido.SemCobertura(),
		})
	}

	return resposta
}

func NovosAtendimentosResponse(atendimentos []domainimportacao.AtendimentoDoPedido) []AtendimentoResponse {
	respostas := make([]AtendimentoResponse, 0, len(atendimentos))
	for _, atendimento := range atendimentos {
		respostas = append(respostas, AtendimentoResponse{
			ImportacaoID:     atendimento.ImportacaoID,
			ImportacaoNumero: atendimento.ImportacaoNumero,
			ImportacaoStatus: atendimento.ImportacaoStatus.String(),
			ChegadaPrevista:  formatarDataOpcional(atendimento.ChegadaPrevista),
			RecebidaEm:       formatarDataOpcional(atendimento.RecebidaEm),
			PedidoItemID:     atendimento.PedidoItemID,
			ProdutoID:        atendimento.ProdutoID,
			Quantidade:       atendimento.Quantidade,
			Reservada:        atendimento.Reservada,
		})
	}

	return respostas
}
