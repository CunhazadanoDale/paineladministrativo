package divida

import (
	"fmt"
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type ContaGerada struct {
	Solicitacao *domainsolicitacao.Solicitacao
	Historico   *domainsolicitacao.Historico
}

type Geracao struct {
	Parcelas []*Parcela
	Contas   []ContaGerada
}

func (g *Geracao) Vazia() bool {
	return g == nil || len(g.Parcelas) == 0
}

func (g *Geracao) Detalhe() string {
	numeros := ""
	for i, parcela := range g.Parcelas {
		if i > 0 {
			numeros += ", "
		}
		numeros += fmt.Sprint(parcela.Numero)
	}

	return fmt.Sprintf("%d contas a pagar das parcelas %s", len(g.Contas), numeros)
}

func (d *Divida) GerarContasAte(limite time.Time, usuarioID uuid.UUID, agora time.Time) (*Geracao, error) {
	geracao := &Geracao{}
	if d.Status != StatusAtiva {
		return geracao, nil
	}

	total := d.TotalDeParcelas()
	for _, parcela := range d.vigentes() {
		if parcela.Status != StatusParcelaPrevista || parcela.Vencimento.After(limite) {
			continue
		}

		contas, err := d.contasDaParcela(parcela, total, usuarioID, agora)
		if err != nil {
			return nil, err
		}
		for _, conta := range contas {
			estado := &Conta{SolicitacaoID: conta.Solicitacao.ID, Status: conta.Solicitacao.Status}
			if conta.Solicitacao.Origem == domainsolicitacao.OrigemDividaJuros {
				parcela.ContaJuros = estado
			} else {
				parcela.ContaAmortizacao = estado
			}
		}

		gerada := agora
		parcela.Status = StatusParcelaGerada
		parcela.GeradaEm = &gerada
		parcela.AtualizadoEm = agora
		geracao.Parcelas = append(geracao.Parcelas, parcela)
		geracao.Contas = append(geracao.Contas, contas...)
	}
	if !geracao.Vazia() {
		d.AtualizadoEm = agora
	}

	return geracao, nil
}

func (d *Divida) contasDaParcela(parcela *Parcela, total int, usuarioID uuid.UUID, agora time.Time) ([]ContaGerada, error) {
	componentes := []struct {
		origem    domainsolicitacao.Origem
		categoria uuid.UUID
		valor     int64
		nome      string
	}{
		{domainsolicitacao.OrigemDividaAmortizacao, d.CategoriaAmortizacaoID, parcela.AmortizacaoCentavos, "amortização"},
		{domainsolicitacao.OrigemDividaJuros, d.CategoriaJurosID, parcela.JurosCentavos, "juros"},
	}

	contas := make([]ContaGerada, 0, len(componentes))
	for _, componente := range componentes {
		if componente.valor == 0 {
			continue
		}

		fornecedorID := d.FornecedorID
		solicitacao, historico, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
			EmpresaID:             d.EmpresaID,
			SolicitanteID:         usuarioID,
			FornecedorID:          &fornecedorID,
			CategoriaFinanceiraID: componente.categoria,
			Origem:                componente.origem,
			OrigemID:              parcela.ID,
			ValorCentavos:         componente.valor,
			Vencimento:            parcela.Vencimento,
			Observacao:            fmt.Sprintf("dívida %s — parcela %d/%d — %s", d.Descricao, parcela.Numero, total, componente.nome),
		}, agora)
		if err != nil {
			return nil, err
		}
		contas = append(contas, ContaGerada{Solicitacao: solicitacao, Historico: historico})
	}

	return contas, nil
}
