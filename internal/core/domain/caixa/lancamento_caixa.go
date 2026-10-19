package caixa

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type LancamentoCaixa struct {
	ID                    uuid.UUID
	EmpresaID             uuid.UUID
	ContaBancariaID       uuid.UUID
	Data                  time.Time
	ValorCentavos         int64
	CategoriaFinanceiraID uuid.UUID
	Competencia           time.Time
	Descricao             string
	Estornado             bool
	UsuarioID             uuid.UUID
	CriadoEm              time.Time
}

type DadosClassificacao struct {
	Lancamento *LancamentoExtrato
	EmpresaID  uuid.UUID
	Categoria  *domainfinanceiro.CategoriaFinanceira
	Descricao  string
	UsuarioID  uuid.UUID
	Agora      time.Time
}

type Classificacao struct {
	Lancamento  *LancamentoCaixa
	Conciliacao *Conciliacao
}

func NovaClassificacao(dados DadosClassificacao) (*Classificacao, error) {
	linha := dados.Lancamento
	if linha == nil {
		return nil, domain.ErroValidacao("lançamento do extrato não informado")
	}
	if dados.EmpresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa da conta não informada")
	}
	if err := exigirCategoriaLancavel(dados.Categoria); err != nil {
		return nil, err
	}

	natureza := domainfinanceiro.NaturezaDespesa
	if linha.Credito() {
		natureza = domainfinanceiro.NaturezaReceita
	}
	if dados.Categoria.Natureza != natureza {
		return nil, domain.ErroValidacao("linha de " + rotuloDoSinal(linha) + " só recebe categoria de " + natureza.String() +
			"; transferência entre contas tem rota própria")
	}

	lancamento := &LancamentoCaixa{
		ID:                    uuid.New(),
		EmpresaID:             dados.EmpresaID,
		ContaBancariaID:       linha.ContaBancariaID,
		Data:                  linha.Data,
		ValorCentavos:         linha.ValorCentavos,
		CategoriaFinanceiraID: dados.Categoria.ID,
		Competencia:           CompetenciaDe(linha.Data),
		Descricao:             descricaoOuDaLinha(dados.Descricao, linha),
		UsuarioID:             dados.UsuarioID,
		CriadoEm:              dados.Agora,
	}

	conciliacao, err := NovaConciliacao(DadosConciliacao{
		Lancamento: linha,
		Movimentos: []*MovimentoSistema{{
			Tipo:            MovimentoLancamentoCaixa,
			ID:              lancamento.ID,
			ContaBancariaID: lancamento.ContaBancariaID,
			EmpresaID:       lancamento.EmpresaID,
			Data:            lancamento.Data,
			ValorCentavos:   lancamento.ValorCentavos,
			Descricao:       lancamento.Descricao,
			OrigemID:        lancamento.ID,
		}},
		Modo:      ConciliacaoClassificacao,
		UsuarioID: dados.UsuarioID,
		Agora:     dados.Agora,
	})
	if err != nil {
		return nil, err
	}

	return &Classificacao{Lancamento: lancamento, Conciliacao: conciliacao}, nil
}

func CompetenciaDe(data time.Time) time.Time {
	ano, mes, _ := data.Date()

	return time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)
}

func exigirCategoriaLancavel(categoria *domainfinanceiro.CategoriaFinanceira) error {
	if categoria == nil {
		return domain.ErroValidacao("categoria financeira não encontrada")
	}
	if !categoria.Lancavel() {
		return domain.ErroValidacao("categoria financeira precisa ser uma subcategoria ativa")
	}

	return nil
}

func descricaoOuDaLinha(descricao string, linha *LancamentoExtrato) string {
	if texto := textoLimitado(descricao, tamanhoMaximoDescricao); texto != "" {
		return texto
	}

	return linha.Descricao
}

func rotuloDoSinal(linha *LancamentoExtrato) string {
	if linha.Credito() {
		return "crédito"
	}

	return "débito"
}
