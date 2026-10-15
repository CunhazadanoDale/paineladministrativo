package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func (c *cenarioBancoClassificacao) inserirConta(t *testing.T, empresaID uuid.UUID, conta, apelido string) uuid.UUID {
	t.Helper()

	nova := contaDeDominio(t, empresaID, domainfinanceiro.BancoDoBrasil, "1234", conta, apelido)
	if err := postgres.NewContaBancariaRepository(c.banco).Criar(context.Background(), nova); err != nil {
		t.Fatalf("criação da conta falhou: %v", err)
	}

	return nova.ID
}

func (c *cenarioBancoClassificacao) aprovadaClassificada(t *testing.T) *domainsolicitacao.Solicitacao {
	t.Helper()

	solicitacao := c.novaSolicitacao(t)
	solicitacao.CategoriaFinanceiraID = &c.despesaID
	if err := c.gravar(t, solicitacao); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	statusAnterior := solicitacao.Status
	if err := solicitacao.Aprovar(c.terceiroID, time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoAprovada, "solicitação aprovada",
	)
	if atualizado, err := c.solicitacoes.AtualizarStatus(context.Background(), solicitacao, historico); err != nil || !atualizado {
		t.Fatalf("aprovação no banco = %v, %v", atualizado, err)
	}

	return solicitacao
}

func (c *cenarioBancoClassificacao) pagar(t *testing.T, solicitacao *domainsolicitacao.Solicitacao, contaID uuid.UUID, comprovanteID *uuid.UUID) (*domainsolicitacao.Pagamento, error) {
	t.Helper()

	statusAnterior := solicitacao.Status
	copia := *solicitacao
	if err := copia.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Fatalf("pagamento no domínio falhou: %v", err)
	}
	pagamento, err := domainsolicitacao.NovoPagamento(
		copia.ID, contaID, c.terceiroID, domainsolicitacao.ValorDe(148500), comprovanteID, time.Now().UTC().Add(-time.Hour), time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("pagamento de teste inválido: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		copia.ID, c.terceiroID, &statusAnterior, copia.Status, domainsolicitacao.EventoPaga, "pagamento registrado",
	)

	if err := c.solicitacoes.CriarPagamento(context.Background(), &copia, pagamento, historico); err != nil {
		return nil, err
	}
	*solicitacao = copia

	return pagamento, nil
}

func (c *cenarioBancoClassificacao) estornar(t *testing.T, solicitacao *domainsolicitacao.Solicitacao, pagamento *domainsolicitacao.Pagamento) error {
	t.Helper()

	motivo, err := domainsolicitacao.NovoMotivoEstorno("pago em duplicidade")
	if err != nil {
		t.Fatalf("motivo de teste inválido: %v", err)
	}

	versaoLida := solicitacao.Versao
	statusAnterior := solicitacao.Status
	copia := *solicitacao
	if err := copia.Estornar(time.Now().UTC()); err != nil {
		t.Fatalf("estorno no domínio falhou: %v", err)
	}
	pagamentoCopia := *pagamento
	if _, err := pagamentoCopia.Estornar(motivo, c.terceiroID, time.Time{}, time.Now().UTC()); err != nil {
		t.Fatalf("estorno do pagamento no domínio falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		copia.ID, c.terceiroID, &statusAnterior, copia.Status, domainsolicitacao.EventoEstornada, "pagamento estornado",
	)

	if err := c.solicitacoes.Estornar(context.Background(), &copia, versaoLida, &pagamentoCopia, historico); err != nil {
		return err
	}
	*solicitacao = copia

	return nil
}

func TestCriarPagamentoGravaContaUsuarioEComprovante(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	contaID := c.inserirConta(t, c.empresaID, "54321", "Movimento")
	solicitacao := c.aprovadaClassificada(t)
	arquivo := c.criarArquivo(t, "comprovante.pdf", "application/pdf")
	aindaAprovada := *solicitacao

	pagamento, err := c.pagar(t, solicitacao, contaID, &arquivo.ID)
	if err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}

	salvo, err := c.solicitacoes.ObterPagamento(ctx, solicitacao.ID)
	if err != nil || salvo == nil {
		t.Fatalf("pagamento vigente = %v, %v", salvo, err)
	}
	if salvo.ID != pagamento.ID || salvo.Valor.Centavos() != 148500 || salvo.Estornado || salvo.Estorno != nil {
		t.Errorf("salvo = %+v", salvo)
	}
	if *salvo.ContaBancariaID != contaID || *salvo.UsuarioID != c.terceiroID || *salvo.ComprovanteArquivoID != arquivo.ID {
		t.Errorf("conta = %v, usuário = %v, comprovante = %v", salvo.ContaBancariaID, salvo.UsuarioID, salvo.ComprovanteArquivoID)
	}
	if paga := c.obter(t, solicitacao.ID); paga.Status != domainsolicitacao.StatusPago {
		t.Errorf("status = %q, esperado pago", paga.Status)
	}

	if _, err := c.pagar(t, &aindaAprovada, contaID, nil); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("segundo pagamento com a leitura antiga = %v, esperado conflito", err)
	}
}

func TestCriarPagamentoRecusaContaInvalidaNaTransacao(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	outraEmpresa := inserirEmpresa(t, c.banco, "11.444.777/0001-61", "Red Diamond")
	daOutra := c.inserirConta(t, outraEmpresa, "11111", "Red")
	inativa := c.inserirConta(t, c.empresaID, "22222", "Antiga")
	if _, err := c.banco.ExecContext(ctx, `UPDATE conta_bancaria SET ativa = false WHERE id = $1`, inativa); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}
	valida := c.inserirConta(t, c.empresaID, "33333", "Movimento")

	casos := map[string]struct {
		contaID  uuid.UUID
		preparar func()
		mensagem string
	}{
		"inexistente":      {uuid.New(), func() {}, domainsolicitacao.ErroContaBancariaNaoEncontrada().Error()},
		"de outra empresa": {daOutra, func() {}, domainsolicitacao.ErroContaBancariaDeOutraEmpresa().Error()},
		"inativa":          {inativa, func() {}, domainsolicitacao.ErroContaBancariaInativa().Error()},
		"empresa desativada": {valida, func() {
			if _, err := c.banco.ExecContext(ctx, `UPDATE empresa SET ativa = false WHERE id = $1`, c.empresaID); err != nil {
				t.Fatalf("desativação falhou: %v", err)
			}
		}, domainsolicitacao.ErroEmpresaInativaNoPagamento().Error()},
	}

	for _, nome := range []string{"inexistente", "de outra empresa", "inativa", "empresa desativada"} {
		caso := casos[nome]
		solicitacao := c.aprovadaClassificada(t)
		caso.preparar()

		_, err := c.pagar(t, solicitacao, caso.contaID, nil)
		if !errors.Is(err, domain.ErrValidacao) || err.Error() != caso.mensagem {
			t.Errorf("%s = %v, esperado %q", nome, err, caso.mensagem)
		}
		if gravada := c.obter(t, solicitacao.ID); gravada.Status != domainsolicitacao.StatusAprovado {
			t.Errorf("%s: pagamento recusado mudou o status para %q", nome, gravada.Status)
		}
	}
}

func TestEstornarReabreASolicitacaoEPermitePagarDeNovo(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	contaID := c.inserirConta(t, c.empresaID, "54321", "Movimento")
	solicitacao := c.aprovadaClassificada(t)

	primeiro, err := c.pagar(t, solicitacao, contaID, nil)
	if err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}
	if err := c.estornar(t, solicitacao, primeiro); err != nil {
		t.Fatalf("estorno falhou: %v", err)
	}

	reaberta := c.obter(t, solicitacao.ID)
	if reaberta.Status != domainsolicitacao.StatusAprovado || reaberta.Versao != solicitacao.Versao {
		t.Errorf("reaberta = status %q, versão %d (esperado aprovado, %d)", reaberta.Status, reaberta.Versao, solicitacao.Versao)
	}
	if vigente, err := c.solicitacoes.ObterPagamento(ctx, solicitacao.ID); err != nil || vigente != nil {
		t.Errorf("pagamento vigente depois do estorno = %v, %v; esperado nenhum", vigente, err)
	}

	segundo, err := c.pagar(t, reaberta, contaID, nil)
	if err != nil {
		t.Fatalf("novo pagamento depois do estorno falhou: %v", err)
	}

	pagamentos, err := c.solicitacoes.ListarPagamentos(ctx, solicitacao.ID)
	if err != nil || len(pagamentos) != 2 {
		t.Fatalf("pagamentos = %+v, %v; esperado dois", pagamentos, err)
	}
	if pagamentos[0].ID != primeiro.ID || !pagamentos[0].Estornado || pagamentos[0].Estorno == nil {
		t.Errorf("primeiro = %+v, esperado estornado com o estorno", pagamentos[0])
	}
	if pagamentos[0].Estorno.Motivo.Texto() != "pago em duplicidade" || pagamentos[0].Estorno.UsuarioID != c.terceiroID {
		t.Errorf("estorno = %+v", pagamentos[0].Estorno)
	}
	if pagamentos[1].ID != segundo.ID || pagamentos[1].Estornado || pagamentos[1].Estorno != nil {
		t.Errorf("segundo = %+v, esperado vigente", pagamentos[1])
	}

	eventos, err := c.solicitacoes.ListarHistorico(ctx, solicitacao.ID)
	if err != nil || eventos[1].Evento != domainsolicitacao.EventoEstornada || *eventos[1].DeStatus != domainsolicitacao.StatusPago ||
		eventos[1].ParaStatus != domainsolicitacao.StatusAprovado {
		t.Errorf("histórico = %+v, %v; esperado estornada de pago para aprovado antes do último pagamento", eventos, err)
	}
}

func TestEstornarComVersaoVelhaOuPagamentoJaEstornadoNaoGrava(t *testing.T) {
	c := cenarioClassificacao(t)
	contaID := c.inserirConta(t, c.empresaID, "54321", "Movimento")
	solicitacao := c.aprovadaClassificada(t)

	pagamento, err := c.pagar(t, solicitacao, contaID, nil)
	if err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}

	velha := *solicitacao
	velha.Versao--
	if err := c.estornar(t, &velha, pagamento); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}
	if gravada := c.obter(t, solicitacao.ID); gravada.Status != domainsolicitacao.StatusPago {
		t.Errorf("estorno recusado mudou o status para %q", gravada.Status)
	}

	if err := c.estornar(t, solicitacao, pagamento); err != nil {
		t.Fatalf("estorno falhou: %v", err)
	}

	if _, err := c.banco.ExecContext(context.Background(), `UPDATE solicitacao SET status = 'pago' WHERE id = $1`, solicitacao.ID); err != nil {
		t.Fatalf("preparação falhou: %v", err)
	}
	solicitacao.Status = domainsolicitacao.StatusPago
	if err := c.estornar(t, solicitacao, pagamento); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("pagamento já estornado = %v, esperado conflito", err)
	}
}
