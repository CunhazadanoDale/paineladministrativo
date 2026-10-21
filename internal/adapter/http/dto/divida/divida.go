package divida

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/divida"
	"github.com/google/uuid"
)

const formatoData = "2006-01-02"

func ParaData(valor, campo string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, nil
	}
	if data, err := time.Parse(formatoData, valor); err == nil {
		return data, nil
	}

	return time.Time{}, domain.ErroValidacao(campo + " deve estar no formato AAAA-MM-DD")
}

func paraDataOpcional(valor *string, campo string) (*time.Time, error) {
	if valor == nil {
		return nil, nil
	}

	data, err := ParaData(*valor, campo)
	if err != nil || data.IsZero() {
		return nil, err
	}

	return &data, nil
}

func formatarDataOpcional(valor *time.Time) *string {
	if valor == nil {
		return nil
	}

	texto := valor.Format(formatoData)

	return &texto
}

type DadosInformativosRequest struct {
	CredorTipo             string     `json:"credor_tipo"`
	Descricao              string     `json:"descricao"`
	Contrato               *string    `json:"contrato"`
	ValorOriginalCentavos  *int64     `json:"valor_original_centavos"`
	ContratadaEm           *string    `json:"contratada_em"`
	TaxaBP                 *int       `json:"taxa_bp"`
	TaxaPeriodo            string     `json:"taxa_periodo"`
	Indexador              string     `json:"indexador"`
	ContaLiberacaoID       *uuid.UUID `json:"conta_liberacao_id"`
	LiberadaEm             *string    `json:"liberada_em"`
	CategoriaAmortizacaoID uuid.UUID  `json:"categoria_amortizacao_id"`
	CategoriaJurosID       uuid.UUID  `json:"categoria_juros_id"`
	Observacoes            string     `json:"observacoes"`
}

func (r DadosInformativosRequest) ParaDados() (portsin.DadosInformativosInput, error) {
	contratadaEm, err := paraDataOpcional(r.ContratadaEm, "data da contratação")
	if err != nil {
		return portsin.DadosInformativosInput{}, err
	}
	liberadaEm, err := paraDataOpcional(r.LiberadaEm, "data da liberação")
	if err != nil {
		return portsin.DadosInformativosInput{}, err
	}

	return portsin.DadosInformativosInput{
		CredorTipo:             r.CredorTipo,
		Descricao:              r.Descricao,
		Contrato:               r.Contrato,
		ValorOriginalCentavos:  r.ValorOriginalCentavos,
		ContratadaEm:           contratadaEm,
		TaxaBP:                 r.TaxaBP,
		TaxaPeriodo:            r.TaxaPeriodo,
		Indexador:              r.Indexador,
		ContaLiberacaoID:       r.ContaLiberacaoID,
		LiberadaEm:             liberadaEm,
		CategoriaAmortizacaoID: r.CategoriaAmortizacaoID,
		CategoriaJurosID:       r.CategoriaJurosID,
		Observacoes:            r.Observacoes,
	}, nil
}

type ParcelaRequest struct {
	Vencimento          string `json:"vencimento"`
	AmortizacaoCentavos int64  `json:"amortizacao_centavos"`
	JurosCentavos       int64  `json:"juros_centavos"`
}

type NovaDividaRequest struct {
	DadosInformativosRequest
	FornecedorID         uuid.UUID        `json:"fornecedor_id"`
	SaldoInicialCentavos int64            `json:"saldo_inicial_centavos"`
	DataSaldo            string           `json:"data_saldo"`
	Parcelas             []ParcelaRequest `json:"parcelas"`
}

func (r NovaDividaRequest) ParaNova() (portsin.NovaDividaInput, error) {
	dados, err := r.ParaDados()
	if err != nil {
		return portsin.NovaDividaInput{}, err
	}
	dataSaldo, err := ParaData(r.DataSaldo, "data do saldo")
	if err != nil {
		return portsin.NovaDividaInput{}, err
	}

	parcelas := make([]portsin.ParcelaInput, 0, len(r.Parcelas))
	for _, parcela := range r.Parcelas {
		vencimento, err := ParaData(parcela.Vencimento, "vencimento da parcela")
		if err != nil {
			return portsin.NovaDividaInput{}, err
		}
		parcelas = append(parcelas, portsin.ParcelaInput{
			Vencimento:          vencimento,
			AmortizacaoCentavos: parcela.AmortizacaoCentavos,
			JurosCentavos:       parcela.JurosCentavos,
		})
	}

	return portsin.NovaDividaInput{
		FornecedorID:         r.FornecedorID,
		Dados:                dados,
		SaldoInicialCentavos: r.SaldoInicialCentavos,
		DataSaldo:            dataSaldo,
		Parcelas:             parcelas,
	}, nil
}

type CriarDividaRequest struct {
	NovaDividaRequest
	EmpresaID uuid.UUID `json:"empresa_id"`
}

func (r CriarDividaRequest) ParaInput(usuarioID uuid.UUID) (portsin.CriarDividaInput, error) {
	nova, err := r.ParaNova()
	if err != nil {
		return portsin.CriarDividaInput{}, err
	}

	return portsin.CriarDividaInput{NovaDividaInput: nova, UsuarioID: usuarioID, EmpresaID: r.EmpresaID}, nil
}

type RenegociarRequest struct {
	Versao int               `json:"versao"`
	Motivo string            `json:"motivo"`
	Nova   NovaDividaRequest `json:"nova"`
}

type VersaoRequest struct {
	Versao int `json:"versao"`
}

type AlterarDividaRequest struct {
	DadosInformativosRequest
	Versao int `json:"versao"`
}

type AlterarParcelaRequest struct {
	Versao        int    `json:"versao"`
	Vencimento    string `json:"vencimento"`
	JurosCentavos int64  `json:"juros_centavos"`
}

type CancelarRequest struct {
	Versao int    `json:"versao"`
	Motivo string `json:"motivo"`
}

type ContaResponse struct {
	SolicitacaoID uuid.UUID `json:"solicitacao_id"`
	Status        string    `json:"status"`
}

type ParcelaResponse struct {
	ID                  uuid.UUID      `json:"id"`
	Numero              int            `json:"numero"`
	Vencimento          string         `json:"vencimento"`
	AmortizacaoCentavos int64          `json:"amortizacao_centavos"`
	JurosCentavos       int64          `json:"juros_centavos"`
	Status              string         `json:"status"`
	Situacao            string         `json:"situacao"`
	GeradaEm            *time.Time     `json:"gerada_em"`
	SubstitutaID        *uuid.UUID     `json:"substituta_id"`
	ContaAmortizacao    *ContaResponse `json:"conta_amortizacao"`
	ContaJuros          *ContaResponse `json:"conta_juros"`
}

type DividaResponse struct {
	ID                     uuid.UUID  `json:"id"`
	EmpresaID              uuid.UUID  `json:"empresa_id"`
	CredorTipo             string     `json:"credor_tipo"`
	FornecedorID           uuid.UUID  `json:"fornecedor_id"`
	Descricao              string     `json:"descricao"`
	Contrato               *string    `json:"contrato"`
	ValorOriginalCentavos  *int64     `json:"valor_original_centavos"`
	ContratadaEm           *string    `json:"contratada_em"`
	TaxaBP                 *int       `json:"taxa_bp"`
	TaxaPeriodo            *string    `json:"taxa_periodo"`
	Indexador              string     `json:"indexador"`
	SaldoInicialCentavos   int64      `json:"saldo_inicial_centavos"`
	DataSaldo              string     `json:"data_saldo"`
	ContaLiberacaoID       *uuid.UUID `json:"conta_liberacao_id"`
	LiberadaEm             *string    `json:"liberada_em"`
	CategoriaAmortizacaoID uuid.UUID  `json:"categoria_amortizacao_id"`
	CategoriaJurosID       uuid.UUID  `json:"categoria_juros_id"`
	DividaOrigemID         *uuid.UUID `json:"divida_origem_id"`
	Status                 string     `json:"status"`
	Motivo                 *string    `json:"motivo"`
	Observacoes            string     `json:"observacoes"`
	UsuarioID              uuid.UUID  `json:"usuario_id"`
	Versao                 int        `json:"versao"`
	CriadoEm               time.Time  `json:"criado_em"`
	AtualizadoEm           time.Time  `json:"atualizado_em"`
}

type DividaDetalheResponse struct {
	DividaResponse
	SaldoDevedorCentavos int64             `json:"saldo_devedor_centavos"`
	Quitada              bool              `json:"quitada"`
	Parcelas             []ParcelaResponse `json:"parcelas"`
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

type GeracaoDaDividaResponse struct {
	DividaID uuid.UUID `json:"divida_id"`
	Parcelas int       `json:"parcelas"`
	Contas   int       `json:"contas"`
}

type DividaRecusadaResponse struct {
	DividaID uuid.UUID `json:"divida_id"`
	Motivo   string    `json:"motivo"`
}

type ResultadoGeracaoResponse struct {
	Competencia string                    `json:"competencia"`
	Geradas     []GeracaoDaDividaResponse `json:"geradas"`
	Recusadas   []DividaRecusadaResponse  `json:"recusadas"`
}

func NovaDividaResponse(divida *domaindivida.Divida) DividaResponse {
	resposta := DividaResponse{
		ID:                     divida.ID,
		EmpresaID:              divida.EmpresaID,
		CredorTipo:             divida.CredorTipo.String(),
		FornecedorID:           divida.FornecedorID,
		Descricao:              divida.Descricao,
		Contrato:               divida.Contrato,
		ValorOriginalCentavos:  divida.ValorOriginalCentavos,
		ContratadaEm:           formatarDataOpcional(divida.ContratadaEm),
		Indexador:              divida.Indexador.String(),
		SaldoInicialCentavos:   divida.SaldoInicialCentavos,
		DataSaldo:              divida.DataSaldo.Format(formatoData),
		ContaLiberacaoID:       divida.ContaLiberacaoID,
		LiberadaEm:             formatarDataOpcional(divida.LiberadaEm),
		CategoriaAmortizacaoID: divida.CategoriaAmortizacaoID,
		CategoriaJurosID:       divida.CategoriaJurosID,
		DividaOrigemID:         divida.DividaOrigemID,
		Status:                 divida.Status.String(),
		Motivo:                 divida.Motivo,
		Observacoes:            divida.Observacoes,
		UsuarioID:              divida.UsuarioID,
		Versao:                 divida.Versao,
		CriadoEm:               divida.CriadoEm,
		AtualizadoEm:           divida.AtualizadoEm,
	}
	if divida.Taxa != nil {
		pontosBase := divida.Taxa.PontosBase()
		periodo := divida.Taxa.Periodo().String()
		resposta.TaxaBP = &pontosBase
		resposta.TaxaPeriodo = &periodo
	}

	return resposta
}

func NovasDividaResponses(dividas []*domaindivida.Divida) []DividaResponse {
	respostas := make([]DividaResponse, 0, len(dividas))
	for _, divida := range dividas {
		respostas = append(respostas, NovaDividaResponse(divida))
	}

	return respostas
}

func NovaDividaDetalheResponse(visao *portsin.Visao) DividaDetalheResponse {
	parcelas := make([]ParcelaResponse, 0, len(visao.Divida.Parcelas))
	for _, parcela := range visao.Divida.Parcelas {
		parcelas = append(parcelas, ParcelaResponse{
			ID:                  parcela.ID,
			Numero:              parcela.Numero,
			Vencimento:          parcela.Vencimento.Format(formatoData),
			AmortizacaoCentavos: parcela.AmortizacaoCentavos,
			JurosCentavos:       parcela.JurosCentavos,
			Status:              parcela.Status.String(),
			Situacao:            parcela.Situacao(visao.Hoje).String(),
			GeradaEm:            parcela.GeradaEm,
			SubstitutaID:        parcela.SubstitutaID,
			ContaAmortizacao:    novaContaResponse(parcela.ContaAmortizacao),
			ContaJuros:          novaContaResponse(parcela.ContaJuros),
		})
	}

	return DividaDetalheResponse{
		DividaResponse:       NovaDividaResponse(visao.Divida),
		SaldoDevedorCentavos: visao.Divida.SaldoDevedor(),
		Quitada:              visao.Divida.Quitada(),
		Parcelas:             parcelas,
	}
}

func novaContaResponse(conta *domaindivida.Conta) *ContaResponse {
	if conta == nil {
		return nil
	}

	return &ContaResponse{SolicitacaoID: conta.SolicitacaoID, Status: conta.Status.String()}
}

func NovoHistoricoResponses(itens []*domaindivida.Historico) []HistoricoResponse {
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

func NovoResultadoGeracaoResponse(resultado portsin.ResultadoGeracao) ResultadoGeracaoResponse {
	resposta := ResultadoGeracaoResponse{
		Competencia: resultado.Competencia,
		Geradas:     make([]GeracaoDaDividaResponse, 0, len(resultado.Geradas)),
		Recusadas:   make([]DividaRecusadaResponse, 0, len(resultado.Recusadas)),
	}
	for _, gerada := range resultado.Geradas {
		resposta.Geradas = append(resposta.Geradas, GeracaoDaDividaResponse{
			DividaID: gerada.DividaID, Parcelas: gerada.Parcelas, Contas: gerada.Contas,
		})
	}
	for _, recusada := range resultado.Recusadas {
		resposta.Recusadas = append(resposta.Recusadas, DividaRecusadaResponse{DividaID: recusada.DividaID, Motivo: recusada.Motivo})
	}

	return resposta
}
