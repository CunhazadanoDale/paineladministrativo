package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portscomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

var (
	_ portscomissao.PedidoRepository        = (*postgres.PedidoRepository)(nil)
	_ portscomissao.RepresentanteRepository = (*postgres.RepresentanteRepository)(nil)
	_ portscomissao.EmpresaRepository       = (*postgres.EmpresaRepository)(nil)
)

type cenarioBancoComissoes struct {
	*cenarioBancoPedidos
	comissoes *postgres.ComissaoRepository
	carlos    *domaincliente.Representante
	beatriz   *domaincliente.Representante
}

func novoCenarioComissoes(t *testing.T) *cenarioBancoComissoes {
	t.Helper()

	c := &cenarioBancoComissoes{
		cenarioBancoPedidos: novoCenarioPedidos(t),
	}
	c.comissoes = postgres.NewComissaoRepository(c.banco)
	c.carlos = c.novoRepresentante(t, "Carlos", 250)
	c.beatriz = c.novoRepresentante(t, "Beatriz", 400)
	c.entrada(t, c.produto, c.galpao, 50)

	return c
}

func (c *cenarioBancoComissoes) novoRepresentante(t *testing.T, nome string, prontaEntrega int) *domaincliente.Representante {
	t.Helper()

	representante, err := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{
		Nome:                  nome,
		ComissaoProntaEntrega: domaincliente.PercentualDe(prontaEntrega),
		ComissaoImportacao:    domaincliente.PercentualDe(prontaEntrega * 2),
	})
	if err != nil {
		t.Fatalf("representante de teste inválido: %v", err)
	}
	if _, err := postgres.NewRepresentanteRepository(c.banco).Criar(c.ctx, representante); err != nil {
		t.Fatalf("criação do representante falhou: %v", err)
	}

	return representante
}

func (c *cenarioBancoComissoes) pedidoConfirmado(t *testing.T, representante *domaincliente.Representante) *domainpedido.Pedido {
	t.Helper()

	dados := c.dados(domainpedido.ModeloProntaEntrega)
	if representante != nil {
		dados.RepresentanteID = &representante.ID
	}
	pedido := c.novoPedido(t, c.mia, dados)
	c.confirmarReservando(t, pedido)

	return c.obter(t, pedido.ID)
}

func (c *cenarioBancoComissoes) saida(t *testing.T, pedido *domainpedido.Pedido, quantidade int) *domainpedido.Entrega {
	t.Helper()

	entrega := c.novaEntrega(t, c.obter(t, pedido.ID), c.galpao, quantidade)
	pendente, err := c.entregas.Obter(c.ctx, entrega.ID)
	if err != nil || pendente == nil {
		t.Fatalf("obter a entrega = %+v, %v", pendente, err)
	}
	c.gravar(t, c.gravacaoDaSaida(t, c.obter(t, pedido.ID), pendente))

	return pendente
}

func (c *cenarioBancoComissoes) paraApurar(t *testing.T, filtro portscomissao.EntregaFiltro) []*domaincomissao.EntregaParaApurar {
	t.Helper()

	if filtro.Size == 0 {
		filtro.PaginacaoFiltro = domain.PaginacaoFiltro{Page: 1, Size: 100}
	}
	entregas, err := c.comissoes.ListarEntregasParaApurar(c.ctx, filtro)
	if err != nil {
		t.Fatalf("ListarEntregasParaApurar: %v", err)
	}

	return entregas
}

func (c *cenarioBancoComissoes) apurar(t *testing.T, entrega *domaincomissao.EntregaParaApurar, representante *domaincliente.Representante) (*domaincomissao.Apuracao, bool, error) {
	t.Helper()

	apuracao, err := domaincomissao.NovaApuracao(*entrega, representante, c.usuarioID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}
	criada, err := c.comissoes.GravarApuracao(c.ctx, apuracao)

	return apuracao, criada, err
}

func TestComissaoRepositoryDefinePercentualComVersaoAteASaida(t *testing.T) {
	c := novoCenarioComissoes(t)
	pedido := c.pedidoConfirmado(t, c.carlos)

	percentual, err := domaincomissao.NovoPercentualManual(pedido.ID, c.carlos.ID, c.usuarioID, 300, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	if err := c.comissoes.DefinirPercentual(c.ctx, percentual, nil); err != nil {
		t.Fatalf("DefinirPercentual: %v", err)
	}
	if err := c.comissoes.DefinirPercentual(c.ctx, percentual, nil); err == nil {
		t.Fatalf("definir de novo sem versão deveria dar conflito")
	}

	lido, err := c.comissoes.ObterPercentual(c.ctx, pedido.ID)
	if err != nil || lido == nil || lido.Percentual.PontosBase() != 300 || lido.Versao != 1 || lido.Origem != domaincomissao.OrigemPercentualManual {
		t.Fatalf("percentual lido = %+v, %v", lido, err)
	}

	if err := lido.Redefinir(350, c.usuarioID, time.Now().UTC()); err != nil {
		t.Fatalf("Redefinir: %v", err)
	}
	velha := 9
	esperarErro(t, c.comissoes.DefinirPercentual(c.ctx, lido, &velha), domain.ErrConflito, "recarregue")
	lida := 1
	if err := c.comissoes.DefinirPercentual(c.ctx, lido, &lida); err != nil || lido.Versao != 2 {
		t.Fatalf("redefinir = %v, versão %d", err, lido.Versao)
	}

	c.saida(t, pedido, 2)
	esperarErro(t, c.comissoes.DefinirPercentual(c.ctx, lido, intDe(2)), domain.ErrConflito, "primeira saída")

	semRepresentante := c.pedidoConfirmado(t, nil)
	outro, err := domaincomissao.NovoPercentualManual(semRepresentante.ID, c.carlos.ID, c.usuarioID, 300, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	esperarErro(t, c.comissoes.DefinirPercentual(c.ctx, outro, nil), domain.ErrConflito, "recarregue")
}

func TestComissaoRepositoryApuraASaidaUmaVezComOPadraoDoRepresentante(t *testing.T) {
	c := novoCenarioComissoes(t)
	pedido := c.pedidoConfirmado(t, c.carlos)
	semRepresentante := c.pedidoConfirmado(t, nil)
	entrega := c.saida(t, pedido, 5)
	c.saida(t, semRepresentante, 1)
	c.novaEntrega(t, c.obter(t, pedido.ID), c.galpao, 1)

	entregas := c.paraApurar(t, portscomissao.EntregaFiltro{})
	if len(entregas) != 1 || entregas[0].EntregaID != entrega.ID || entregas[0].RepresentanteID != c.carlos.ID ||
		entregas[0].PedidoNumero != pedido.Numero || entregas[0].Percentual != nil {
		t.Fatalf("entregas para apurar = %+v", entregas)
	}
	if len(entregas[0].Itens) != 1 || entregas[0].Itens[0].Quantidade != 5 || entregas[0].Itens[0].PrecoUnitarioCentavos != 1500 {
		t.Fatalf("itens = %+v", entregas[0].Itens)
	}

	apuracao, criada, err := c.apurar(t, entregas[0], c.carlos)
	if err != nil || !criada {
		t.Fatalf("GravarApuracao = %v, %v", criada, err)
	}
	if apuracao.Comissao.BaseCentavos != 7500 || apuracao.Comissao.ValorCentavos != 188 {
		t.Fatalf("comissão = %+v", apuracao.Comissao)
	}

	_, criada, err = c.apurar(t, entregas[0], c.carlos)
	if err != nil || criada {
		t.Fatalf("apurar de novo = %v, %v", criada, err)
	}
	if restantes := c.paraApurar(t, portscomissao.EntregaFiltro{}); len(restantes) != 0 {
		t.Fatalf("a entrega apurada não deveria voltar: %+v", restantes)
	}

	percentual, err := c.comissoes.ObterPercentual(c.ctx, pedido.ID)
	if err != nil || percentual == nil || percentual.Origem != domaincomissao.OrigemPercentualPadrao || percentual.Percentual.PontosBase() != 250 {
		t.Fatalf("percentual gravado na apuração = %+v, %v", percentual, err)
	}

	competencia := domaincomissao.CompetenciaDe(*entrega.SaiuEm)
	listadas, err := c.comissoes.Listar(c.ctx, portscomissao.ComissaoFiltro{RepresentanteID: &c.carlos.ID, Competencia: &competencia})
	if err != nil || len(listadas) != 1 || listadas[0].EntregaID != entrega.ID || listadas[0].ValorCentavos != 188 ||
		listadas[0].Percentual.PontosBase() != 250 || listadas[0].EmpresaID != c.mia {
		t.Fatalf("listadas = %+v, %v", listadas, err)
	}
	if outras, err := c.comissoes.Listar(c.ctx, portscomissao.ComissaoFiltro{RepresentanteID: &c.beatriz.ID}); err != nil || len(outras) != 0 {
		t.Fatalf("comissões da Beatriz = %+v, %v", outras, err)
	}
}

func TestComissaoRepositoryApuraEntregaDeItemBonificadoComBaseZero(t *testing.T) {
	c := novoCenarioComissoes(t)
	dados := c.dados(domainpedido.ModeloProntaEntrega)
	dados.RepresentanteID = &c.carlos.ID
	dados.Itens[0].PrecoUnitarioCentavos = 0
	pedido := c.novoPedido(t, c.mia, dados)
	c.confirmarReservando(t, pedido)
	entrega := c.saida(t, c.obter(t, pedido.ID), 3)

	entregas := c.paraApurar(t, portscomissao.EntregaFiltro{})
	if len(entregas) != 1 || entregas[0].EntregaID != entrega.ID || len(entregas[0].Itens) != 1 || entregas[0].Itens[0].PrecoUnitarioCentavos != 0 {
		t.Fatalf("entregas para apurar = %+v", entregas)
	}

	apuracao, criada, err := c.apurar(t, entregas[0], c.carlos)
	if err != nil || !criada {
		t.Fatalf("GravarApuracao = %v, %v", criada, err)
	}
	if apuracao.Comissao.BaseCentavos != 0 || apuracao.Comissao.ValorCentavos != 0 {
		t.Fatalf("comissão = %+v", apuracao.Comissao)
	}
	if restantes := c.paraApurar(t, portscomissao.EntregaFiltro{}); len(restantes) != 0 {
		t.Fatalf("a entrega bonificada não deveria voltar: %+v", restantes)
	}

	listadas, err := c.comissoes.Listar(c.ctx, portscomissao.ComissaoFiltro{RepresentanteID: &c.carlos.ID})
	if err != nil || len(listadas) != 1 || listadas[0].BaseCentavos != 0 || listadas[0].ValorCentavos != 0 {
		t.Fatalf("listadas = %+v, %v", listadas, err)
	}
}

func TestComissaoRepositoryFiltraEntregasPorEmpresaESaida(t *testing.T) {
	c := novoCenarioComissoes(t)
	entrega := c.saida(t, c.pedidoConfirmado(t, c.carlos), 1)

	if entregas := c.paraApurar(t, portscomissao.EntregaFiltro{Empresa: &c.red}); len(entregas) != 0 {
		t.Fatalf("entregas da Red = %+v", entregas)
	}
	if entregas := c.paraApurar(t, portscomissao.EntregaFiltro{EmpresaIDs: []uuid.UUID{c.red}}); len(entregas) != 0 {
		t.Fatalf("entregas no acesso da Red = %+v", entregas)
	}
	antes := *entrega.SaiuEm
	if entregas := c.paraApurar(t, portscomissao.EntregaFiltro{SaidaAntesDe: &antes}); len(entregas) != 0 {
		t.Fatalf("entregas antes da saída = %+v", entregas)
	}
	depois := antes.AddDate(0, 0, 1)
	if entregas := c.paraApurar(t, portscomissao.EntregaFiltro{Empresa: &c.mia, SaidaAntesDe: &depois}); len(entregas) != 1 {
		t.Fatalf("entregas até o dia seguinte = %+v", entregas)
	}
	if entregas := c.paraApurar(t, portscomissao.EntregaFiltro{IgnorarIDs: []uuid.UUID{entrega.ID}}); len(entregas) != 0 {
		t.Fatalf("a entrega ignorada não deveria voltar: %+v", entregas)
	}
}

func TestComissaoRepositoryApuraComOPercentualManualERecusaDivergencia(t *testing.T) {
	c := novoCenarioComissoes(t)
	pedido := c.pedidoConfirmado(t, c.carlos)
	manual, err := domaincomissao.NovoPercentualManual(pedido.ID, c.carlos.ID, c.usuarioID, 1000, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	if err := c.comissoes.DefinirPercentual(c.ctx, manual, nil); err != nil {
		t.Fatalf("DefinirPercentual: %v", err)
	}
	c.saida(t, pedido, 2)

	entregas := c.paraApurar(t, portscomissao.EntregaFiltro{})
	if len(entregas) != 1 || !entregas[0].Percentual.VigentePara(c.carlos.ID) || entregas[0].Percentual.Percentual.PontosBase() != 1000 {
		t.Fatalf("entrega com percentual = %+v", entregas)
	}

	divergente := *entregas[0]
	divergente.Percentual = &domaincomissao.PercentualPedido{
		PedidoID: pedido.ID, RepresentanteID: c.carlos.ID, Percentual: domaincliente.PercentualDe(50),
		Origem: domaincomissao.OrigemPercentualManual, UsuarioID: c.usuarioID, Versao: 1,
	}
	_, _, err = c.apurar(t, &divergente, nil)
	esperarErro(t, err, domain.ErrConflito, "mudou durante a apuração")

	apuracao, criada, err := c.apurar(t, entregas[0], nil)
	if err != nil || !criada || apuracao.Comissao.ValorCentavos != 300 {
		t.Fatalf("apuração manual = %+v, %v, %v", apuracao.Comissao, criada, err)
	}
}

func TestComissaoRepositorySubstituiPercentualDeOutroRepresentanteNaApuracao(t *testing.T) {
	c := novoCenarioComissoes(t)
	pedido := c.pedidoConfirmado(t, c.carlos)
	manual, err := domaincomissao.NovoPercentualManual(pedido.ID, c.carlos.ID, c.usuarioID, 900, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	if err := c.comissoes.DefinirPercentual(c.ctx, manual, nil); err != nil {
		t.Fatalf("DefinirPercentual: %v", err)
	}
	if _, err := c.banco.ExecContext(c.ctx, `UPDATE pedido_comissao SET representante_id = $1 WHERE pedido_id = $2`, c.beatriz.ID, pedido.ID); err != nil {
		t.Fatalf("trocar o representante da linha: %v", err)
	}
	c.saida(t, pedido, 2)

	entregas := c.paraApurar(t, portscomissao.EntregaFiltro{})
	if len(entregas) != 1 || entregas[0].Percentual.VigentePara(c.carlos.ID) {
		t.Fatalf("a linha de outro representante não vale: %+v", entregas)
	}

	apuracao, criada, err := c.apurar(t, entregas[0], c.carlos)
	if err != nil || !criada || apuracao.Comissao.Percentual.PontosBase() != 250 {
		t.Fatalf("apuração = %+v, %v, %v", apuracao.Comissao, criada, err)
	}

	percentual, err := c.comissoes.ObterPercentual(c.ctx, pedido.ID)
	if err != nil || percentual.RepresentanteID != c.carlos.ID || percentual.Origem != domaincomissao.OrigemPercentualPadrao || percentual.Versao != 2 {
		t.Fatalf("percentual substituído = %+v, %v", percentual, err)
	}
}

func intDe(valor int) *int {
	return &valor
}
