package importacao

import (
	"time"

	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

type ItemRequest struct {
	ProdutoID            uuid.UUID `json:"produto_id"`
	Quantidade           int       `json:"quantidade"`
	ValorMoedaCentesimos int64     `json:"valor_moeda_centesimos"`
}

type DadosImportacaoRequest struct {
	FornecedorID     uuid.UUID     `json:"fornecedor_id"`
	Moeda            string        `json:"moeda"`
	Referencia       *string       `json:"referencia"`
	EmbarquePrevisto *string       `json:"embarque_previsto"`
	ChegadaPrevista  *string       `json:"chegada_prevista"`
	Observacoes      string        `json:"observacoes"`
	Itens            []ItemRequest `json:"itens"`
}

func (r DadosImportacaoRequest) ParaDados() (portsin.DadosImportacaoInput, error) {
	embarquePrevisto, err := paraDataOpcional(r.EmbarquePrevisto, "embarque previsto")
	if err != nil {
		return portsin.DadosImportacaoInput{}, err
	}
	chegadaPrevista, err := paraDataOpcional(r.ChegadaPrevista, "chegada prevista")
	if err != nil {
		return portsin.DadosImportacaoInput{}, err
	}

	itens := make([]portsin.ItemInput, 0, len(r.Itens))
	for _, item := range r.Itens {
		itens = append(itens, portsin.ItemInput{
			ProdutoID:            item.ProdutoID,
			Quantidade:           item.Quantidade,
			ValorMoedaCentesimos: item.ValorMoedaCentesimos,
		})
	}

	return portsin.DadosImportacaoInput{
		FornecedorID:     r.FornecedorID,
		Moeda:            r.Moeda,
		Referencia:       r.Referencia,
		EmbarquePrevisto: embarquePrevisto,
		ChegadaPrevista:  chegadaPrevista,
		Observacoes:      r.Observacoes,
		Itens:            itens,
	}, nil
}

type CriarImportacaoRequest struct {
	DadosImportacaoRequest
	EmpresaID uuid.UUID `json:"empresa_id"`
}

type AlterarImportacaoRequest struct {
	DadosImportacaoRequest
	Versao int `json:"versao"`
}

type EmbarcarRequest struct {
	Versao            int     `json:"versao"`
	EmbarcadaEm       string  `json:"embarcada_em"`
	ChegadaPrevista   string  `json:"chegada_prevista"`
	DocumentoEmbarque *string `json:"documento_embarque"`
}

type DesembaracarRequest struct {
	Versao          int    `json:"versao"`
	ChegadaEm       string `json:"chegada_em"`
	DesembaracadaEm string `json:"desembaracada_em"`
	Declaracao      string `json:"declaracao"`
}

type CancelarRequest struct {
	Versao int    `json:"versao"`
	Motivo string `json:"motivo"`
}

type ItemResponse struct {
	ID                   uuid.UUID `json:"id"`
	ProdutoID            uuid.UUID `json:"produto_id"`
	Quantidade           int       `json:"quantidade"`
	ValorMoedaCentesimos *int64    `json:"valor_moeda_centesimos,omitempty"`
}

type ImportacaoResponse struct {
	ID                        uuid.UUID `json:"id"`
	EmpresaID                 uuid.UUID `json:"empresa_id"`
	Numero                    int       `json:"numero"`
	FornecedorID              uuid.UUID `json:"fornecedor_id"`
	Referencia                *string   `json:"referencia"`
	Moeda                     string    `json:"moeda"`
	Status                    string    `json:"status"`
	ValorTotalMoedaCentesimos *int64    `json:"valor_total_moeda_centesimos,omitempty"`
	EmbarquePrevisto          *string   `json:"embarque_previsto"`
	EmbarcadaEm               *string   `json:"embarcada_em"`
	ChegadaPrevista           *string   `json:"chegada_prevista"`
	ChegadaEm                 *string   `json:"chegada_em"`
	DesembaracadaEm           *string   `json:"desembaracada_em"`
	DocumentoEmbarque         *string   `json:"documento_embarque"`
	Declaracao                *string   `json:"declaracao"`
	Observacoes               string    `json:"observacoes"`
	CanceladaMotivo           *string   `json:"cancelada_motivo"`
	UsuarioID                 uuid.UUID `json:"usuario_id"`
	Versao                    int       `json:"versao"`
	CriadoEm                  time.Time `json:"criado_em"`
	AtualizadoEm              time.Time `json:"atualizado_em"`
}

type ImportacaoDetalheResponse struct {
	ImportacaoResponse
	Itens []ItemResponse `json:"itens"`
}

type HistoricoResponse struct {
	ID             uuid.UUID `json:"id"`
	UsuarioID      uuid.UUID `json:"usuario_id"`
	Acao           string    `json:"acao"`
	StatusAnterior *string   `json:"status_anterior"`
	StatusNovo     string    `json:"status_novo"`
	Detalhe        string    `json:"detalhe"`
	CriadoEm       time.Time `json:"criado_em"`
}

func NovaImportacaoResponse(item *domainimportacao.Importacao, comValores bool) ImportacaoResponse {
	resposta := ImportacaoResponse{
		ID:                item.ID,
		EmpresaID:         item.EmpresaID,
		Numero:            item.Numero,
		FornecedorID:      item.FornecedorID,
		Referencia:        item.Referencia,
		Moeda:             item.Moeda.String(),
		Status:            item.Status.String(),
		EmbarquePrevisto:  formatarDataOpcional(item.EmbarquePrevisto),
		EmbarcadaEm:       formatarDataOpcional(item.EmbarcadaEm),
		ChegadaPrevista:   formatarDataOpcional(item.ChegadaPrevista),
		ChegadaEm:         formatarDataOpcional(item.ChegadaEm),
		DesembaracadaEm:   formatarDataOpcional(item.DesembaracadaEm),
		DocumentoEmbarque: item.DocumentoEmbarque,
		Declaracao:        item.Declaracao,
		Observacoes:       item.Observacoes,
		CanceladaMotivo:   item.CanceladaMotivo,
		UsuarioID:         item.UsuarioID,
		Versao:            item.Versao,
		CriadoEm:          item.CriadoEm,
		AtualizadoEm:      item.AtualizadoEm,
	}
	if comValores {
		total := item.ValorTotal.Centesimos()
		resposta.ValorTotalMoedaCentesimos = &total
	}

	return resposta
}

func NovaImportacaoDetalheResponse(visao *portsin.Visao) ImportacaoDetalheResponse {
	itens := make([]ItemResponse, 0, len(visao.Importacao.Itens))
	for _, item := range visao.Importacao.Itens {
		resposta := ItemResponse{ID: item.ID, ProdutoID: item.ProdutoID, Quantidade: item.Quantidade}
		if visao.ComValores {
			valor := item.Valor.Centesimos()
			resposta.ValorMoedaCentesimos = &valor
		}
		itens = append(itens, resposta)
	}

	return ImportacaoDetalheResponse{
		ImportacaoResponse: NovaImportacaoResponse(visao.Importacao, visao.ComValores),
		Itens:              itens,
	}
}

func NovaImportacaoResponses(lista *portsin.Lista) []ImportacaoResponse {
	respostas := make([]ImportacaoResponse, 0, len(lista.Importacoes))
	for _, item := range lista.Importacoes {
		respostas = append(respostas, NovaImportacaoResponse(item, lista.ComValores))
	}

	return respostas
}

func NovoHistoricoResponses(itens []*domainimportacao.Historico) []HistoricoResponse {
	respostas := make([]HistoricoResponse, 0, len(itens))
	for _, item := range itens {
		resposta := HistoricoResponse{
			ID:         item.ID,
			UsuarioID:  item.UsuarioID,
			Acao:       item.Acao.String(),
			StatusNovo: item.StatusNovo.String(),
			Detalhe:    item.Detalhe,
			CriadoEm:   item.CriadoEm,
		}
		if item.StatusAnterior != nil {
			anterior := item.StatusAnterior.String()
			resposta.StatusAnterior = &anterior
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}
