package caixa

import (
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

type ClassificarRequest struct {
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	Descricao             string    `json:"descricao"`
}

func (r ClassificarRequest) ParaInput(usuarioID, lancamentoID uuid.UUID) portsin.ClassificarInput {
	return portsin.ClassificarInput{
		UsuarioID:             usuarioID,
		LancamentoID:          lancamentoID,
		CategoriaFinanceiraID: r.CategoriaFinanceiraID,
		Descricao:             r.Descricao,
	}
}

type TransferirRequest struct {
	ContaContraparteID    uuid.UUID `json:"conta_contraparte_id"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	Descricao             string    `json:"descricao"`
}

func (r TransferirRequest) ParaInput(usuarioID, lancamentoID uuid.UUID) portsin.TransferirInput {
	return portsin.TransferirInput{
		UsuarioID:             usuarioID,
		LancamentoID:          lancamentoID,
		ContraparteID:         r.ContaContraparteID,
		CategoriaFinanceiraID: r.CategoriaFinanceiraID,
		Descricao:             r.Descricao,
	}
}

type LancamentoCaixaResponse struct {
	ID                    uuid.UUID `json:"id"`
	EmpresaID             uuid.UUID `json:"empresa_id"`
	ContaBancariaID       uuid.UUID `json:"conta_bancaria_id"`
	Data                  string    `json:"data"`
	ValorCentavos         int64     `json:"valor_centavos"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	Competencia           string    `json:"competencia"`
	Descricao             string    `json:"descricao"`
	Estornado             bool      `json:"estornado"`
	UsuarioID             uuid.UUID `json:"usuario_id"`
	CriadoEm              time.Time `json:"criado_em"`
}

func NovoLancamentoCaixaResponse(lancamento *domaincaixa.LancamentoCaixa) LancamentoCaixaResponse {
	return LancamentoCaixaResponse{
		ID:                    lancamento.ID,
		EmpresaID:             lancamento.EmpresaID,
		ContaBancariaID:       lancamento.ContaBancariaID,
		Data:                  formatarData(lancamento.Data),
		ValorCentavos:         lancamento.ValorCentavos,
		CategoriaFinanceiraID: lancamento.CategoriaFinanceiraID,
		Competencia:           lancamento.Competencia.Format("2006-01"),
		Descricao:             lancamento.Descricao,
		Estornado:             lancamento.Estornado,
		UsuarioID:             lancamento.UsuarioID,
		CriadoEm:              lancamento.CriadoEm,
	}
}

func NovosLancamentoCaixaResponses(lancamentos []*domaincaixa.LancamentoCaixa) []LancamentoCaixaResponse {
	respostas := make([]LancamentoCaixaResponse, 0, len(lancamentos))
	for _, lancamento := range lancamentos {
		respostas = append(respostas, NovoLancamentoCaixaResponse(lancamento))
	}

	return respostas
}

type TransferenciaResponse struct {
	ID                    uuid.UUID `json:"id"`
	ContaOrigemID         uuid.UUID `json:"conta_origem_id"`
	ContaDestinoID        uuid.UUID `json:"conta_destino_id"`
	Data                  string    `json:"data"`
	ValorCentavos         int64     `json:"valor_centavos"`
	EntreEmpresas         bool      `json:"entre_empresas"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	Descricao             string    `json:"descricao"`
	Estornado             bool      `json:"estornado"`
	UsuarioID             uuid.UUID `json:"usuario_id"`
	CriadoEm              time.Time `json:"criado_em"`
}

func NovaTransferenciaResponse(transferencia *domaincaixa.Transferencia) TransferenciaResponse {
	return TransferenciaResponse{
		ID:                    transferencia.ID,
		ContaOrigemID:         transferencia.ContaOrigemID,
		ContaDestinoID:        transferencia.ContaDestinoID,
		Data:                  formatarData(transferencia.Data),
		ValorCentavos:         transferencia.ValorCentavos,
		EntreEmpresas:         transferencia.EntreEmpresas,
		CategoriaFinanceiraID: transferencia.CategoriaFinanceiraID,
		Descricao:             transferencia.Descricao,
		Estornado:             transferencia.Estornado,
		UsuarioID:             transferencia.UsuarioID,
		CriadoEm:              transferencia.CriadoEm,
	}
}

func NovasTransferenciaResponses(transferencias []*domaincaixa.Transferencia) []TransferenciaResponse {
	respostas := make([]TransferenciaResponse, 0, len(transferencias))
	for _, transferencia := range transferencias {
		respostas = append(respostas, NovaTransferenciaResponse(transferencia))
	}

	return respostas
}

type ClassificacaoResponse struct {
	LancamentoCaixa LancamentoCaixaResponse `json:"lancamento_caixa"`
	Conciliacao     ConciliacaoResponse     `json:"conciliacao"`
}

func NovaClassificacaoResponse(classificacao *domaincaixa.Classificacao) ClassificacaoResponse {
	return ClassificacaoResponse{
		LancamentoCaixa: NovoLancamentoCaixaResponse(classificacao.Lancamento),
		Conciliacao:     NovaConciliacaoResponse(classificacao.Conciliacao),
	}
}

type TransferenciaRegistradaResponse struct {
	Transferencia TransferenciaResponse `json:"transferencia"`
	Conciliacao   ConciliacaoResponse   `json:"conciliacao"`
}

func NovaTransferenciaRegistradaResponse(registrada *domaincaixa.TransferenciaRegistrada) TransferenciaRegistradaResponse {
	return TransferenciaRegistradaResponse{
		Transferencia: NovaTransferenciaResponse(registrada.Transferencia),
		Conciliacao:   NovaConciliacaoResponse(registrada.Conciliacao),
	}
}
