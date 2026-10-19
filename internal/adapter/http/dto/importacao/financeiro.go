package importacao

import (
	"time"

	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

type ParcelaRequest struct {
	Versao               int    `json:"versao"`
	Descricao            string `json:"descricao"`
	ValorMoedaCentesimos int64  `json:"valor_moeda_centesimos"`
	VencimentoPrevisto   string `json:"vencimento_previsto"`
}

func (r ParcelaRequest) ParaDados() (portsin.DadosParcelaInput, error) {
	vencimento, err := ParaData(r.VencimentoPrevisto, "vencimento previsto da parcela")
	if err != nil {
		return portsin.DadosParcelaInput{}, err
	}

	return portsin.DadosParcelaInput{
		Descricao:            r.Descricao,
		ValorMoedaCentesimos: r.ValorMoedaCentesimos,
		VencimentoPrevisto:   vencimento,
	}, nil
}

type VersaoRequest struct {
	Versao int `json:"versao"`
}

type CambioRequest struct {
	Versao                int       `json:"versao"`
	CambioMicro           *int64    `json:"cambio_micro"`
	Vencimento            string    `json:"vencimento"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
}

type CustoRequest struct {
	Versao                int       `json:"versao"`
	Tipo                  string    `json:"tipo"`
	Descricao             string    `json:"descricao"`
	FornecedorID          uuid.UUID `json:"fornecedor_id"`
	ValorCentavos         int64     `json:"valor_centavos"`
	Vencimento            string    `json:"vencimento"`
	CompoeCusto           *bool     `json:"compoe_custo"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
}

type ParcelaResponse struct {
	ID                   uuid.UUID  `json:"id"`
	Descricao            string     `json:"descricao"`
	ValorMoedaCentesimos int64      `json:"valor_moeda_centesimos"`
	VencimentoPrevisto   string     `json:"vencimento_previsto"`
	Status               string     `json:"status"`
	CambioMicro          *int64     `json:"cambio_micro"`
	ValorCentavos        *int64     `json:"valor_centavos"`
	Vencimento           *string    `json:"vencimento"`
	SolicitacaoID        *uuid.UUID `json:"solicitacao_id"`
	StatusContaAPagar    *string    `json:"status_conta_a_pagar"`
	UsuarioID            uuid.UUID  `json:"usuario_id"`
	CriadoEm             time.Time  `json:"criado_em"`
	AtualizadoEm         time.Time  `json:"atualizado_em"`
}

type CustoResponse struct {
	ID                    uuid.UUID  `json:"id"`
	Tipo                  string     `json:"tipo"`
	Descricao             string     `json:"descricao"`
	FornecedorID          uuid.UUID  `json:"fornecedor_id"`
	ValorCentavos         int64      `json:"valor_centavos"`
	Vencimento            string     `json:"vencimento"`
	CompoeCusto           bool       `json:"compoe_custo"`
	Posterior             bool       `json:"posterior"`
	CategoriaFinanceiraID uuid.UUID  `json:"categoria_financeira_id"`
	SolicitacaoID         uuid.UUID  `json:"solicitacao_id"`
	StatusContaAPagar     *string    `json:"status_conta_a_pagar"`
	Cancelado             bool       `json:"cancelado"`
	CanceladoEm           *time.Time `json:"cancelado_em"`
	CanceladoPor          *uuid.UUID `json:"cancelado_por"`
	UsuarioID             uuid.UUID  `json:"usuario_id"`
	CriadoEm              time.Time  `json:"criado_em"`
}

type CustoDoItemResponse struct {
	ItemID                  uuid.UUID `json:"item_id"`
	ProdutoID               uuid.UUID `json:"produto_id"`
	Quantidade              int       `json:"quantidade"`
	ValorMoedaCentesimos    int64     `json:"valor_moeda_centesimos"`
	ValorMercadoriaCentavos int64     `json:"valor_mercadoria_centavos"`
	CustosCentavos          int64     `json:"custos_centavos"`
	CustoTotalCentavos      int64     `json:"custo_total_centavos"`
	CustoUnitarioCentavos   *int64    `json:"custo_unitario_centavos"`
}

type CustoPrevistoResponse struct {
	ValorMoedaCentesimos            int64                 `json:"valor_moeda_centesimos"`
	ParcelasFechadasMoedaCentesimos int64                 `json:"parcelas_fechadas_moeda_centesimos"`
	ParcelasFechadasCentavos        int64                 `json:"parcelas_fechadas_centavos"`
	SemCambioFechadoMoedaCentesimos int64                 `json:"sem_cambio_fechado_moeda_centesimos"`
	CambioInformadoMicro            *int64                `json:"cambio_informado_micro"`
	CambioMedioMicro                int64                 `json:"cambio_medio_micro"`
	ValorMercadoriaCentavos         int64                 `json:"valor_mercadoria_centavos"`
	CustosCentavos                  int64                 `json:"custos_centavos"`
	CustosForaDoRateioCentavos      int64                 `json:"custos_fora_do_rateio_centavos"`
	Itens                           []CustoDoItemResponse `json:"itens"`
}

func NovasParcelasResponse(parcelas []*domainimportacao.Parcela) []ParcelaResponse {
	respostas := make([]ParcelaResponse, 0, len(parcelas))
	for _, parcela := range parcelas {
		resposta := ParcelaResponse{
			ID:                   parcela.ID,
			Descricao:            parcela.Descricao,
			ValorMoedaCentesimos: parcela.Valor.Centesimos(),
			VencimentoPrevisto:   parcela.VencimentoPrevisto.Format(formatoData),
			Status:               parcela.Status.String(),
			ValorCentavos:        parcela.ValorCentavos,
			Vencimento:           formatarDataOpcional(parcela.Vencimento),
			SolicitacaoID:        parcela.SolicitacaoID,
			UsuarioID:            parcela.UsuarioID,
			CriadoEm:             parcela.CriadoEm,
			AtualizadoEm:         parcela.AtualizadoEm,
		}
		if parcela.Cambio != nil {
			micro := parcela.Cambio.Micro()
			resposta.CambioMicro = &micro
		}
		if parcela.StatusContaAPagar != nil {
			status := parcela.StatusContaAPagar.String()
			resposta.StatusContaAPagar = &status
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}

func NovosCustosResponse(custos []*domainimportacao.Custo) []CustoResponse {
	respostas := make([]CustoResponse, 0, len(custos))
	for _, custo := range custos {
		resposta := CustoResponse{
			ID:                    custo.ID,
			Tipo:                  custo.Tipo.String(),
			Descricao:             custo.Descricao,
			FornecedorID:          custo.FornecedorID,
			ValorCentavos:         custo.ValorCentavos,
			Vencimento:            custo.Vencimento.Format(formatoData),
			CompoeCusto:           custo.CompoeCusto,
			Posterior:             custo.Posterior,
			CategoriaFinanceiraID: custo.CategoriaFinanceiraID,
			SolicitacaoID:         custo.SolicitacaoID,
			Cancelado:             !custo.Ativo(),
			CanceladoEm:           custo.CanceladoEm,
			CanceladoPor:          custo.CanceladoPor,
			UsuarioID:             custo.UsuarioID,
			CriadoEm:              custo.CriadoEm,
		}
		if custo.StatusContaAPagar != nil {
			status := custo.StatusContaAPagar.String()
			resposta.StatusContaAPagar = &status
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}

func NovoCustoPrevistoResponse(calculo *domainimportacao.CalculoCusto) CustoPrevistoResponse {
	resposta := CustoPrevistoResponse{
		ValorMoedaCentesimos:            calculo.Valor.Centesimos(),
		ParcelasFechadasMoedaCentesimos: calculo.ParcelasFechadas.Centesimos(),
		ParcelasFechadasCentavos:        calculo.ParcelasFechadasCentavos,
		SemCambioFechadoMoedaCentesimos: calculo.SemCambioFechado.Centesimos(),
		CambioMedioMicro:                calculo.CambioMedio.Micro(),
		ValorMercadoriaCentavos:         calculo.ValorMercadoriaCentavos,
		CustosCentavos:                  calculo.CustosCentavos,
		CustosForaDoRateioCentavos:      calculo.CustosForaDoRateioCentavos,
		Itens:                           make([]CustoDoItemResponse, 0, len(calculo.Itens)),
	}
	if calculo.CambioInformado != nil {
		micro := calculo.CambioInformado.Micro()
		resposta.CambioInformadoMicro = &micro
	}
	for _, item := range calculo.Itens {
		resposta.Itens = append(resposta.Itens, CustoDoItemResponse{
			ItemID:                  item.ItemID,
			ProdutoID:               item.ProdutoID,
			Quantidade:              item.Quantidade,
			ValorMoedaCentesimos:    item.Valor.Centesimos(),
			ValorMercadoriaCentavos: item.ValorMercadoriaCentavos,
			CustosCentavos:          item.CustosCentavos,
			CustoTotalCentavos:      item.CustoTotalCentavos,
			CustoUnitarioCentavos:   item.CustoUnitarioCentavos,
		})
	}

	return resposta
}
