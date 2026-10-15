package pedido_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	pedidousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/pedido"
	"github.com/google/uuid"
)

type repoConsultas struct {
	pendencias    *portsout.PendenciaFiltro
	prazos        *portsout.PrazoFiltro
	cancelamentos *portsout.CancelamentoFiltro
}

func (r *repoConsultas) Pendencias(_ context.Context, filtro portsout.PendenciaFiltro) ([]*domainpedido.Pendencia, error) {
	r.pendencias = &filtro
	return nil, nil
}

func (r *repoConsultas) PorPrazo(_ context.Context, filtro portsout.PrazoFiltro) ([]*domainpedido.Pedido, error) {
	r.prazos = &filtro
	return nil, nil
}

func (r *repoConsultas) Cancelamentos(_ context.Context, filtro portsout.CancelamentoFiltro) ([]*domainpedido.Pedido, error) {
	r.cancelamentos = &filtro
	return nil, nil
}

func (c *cenario) consultas() (portsin.ConsultaUseCase, *repoConsultas) {
	repo := &repoConsultas{}

	return pedidousecases.NewConsultaUsecase(repo, repoRepresentantes{c.m}, repoEmpresas{c.m}, repoUsuarios{c.m}, repoCargos{c.m}), repo
}

func TestConsultasRestringemAoAcessoDoUsuario(t *testing.T) {
	c := novoCenario(t)
	usecase, repo := c.consultas()

	if _, err := usecase.Pendencias(c.ctx, portsin.ConsultarPendenciasInput{UsuarioID: c.comercial, ProdutoID: &c.cimento.ID}); err != nil {
		t.Fatalf("Pendencias falhou: %v", err)
	}
	escopo := repo.pendencias.EscopoFiltro
	if len(escopo.EmpresaIDs) != 2 || escopo.RepresentanteID != nil || *repo.pendencias.ProdutoID != c.cimento.ID {
		t.Errorf("filtro do comercial da MIA = %+v", repo.pendencias)
	}

	if _, err := usecase.Pendencias(c.ctx, portsin.ConsultarPendenciasInput{UsuarioID: c.administrador}); err != nil {
		t.Fatalf("Pendencias falhou: %v", err)
	}
	if repo.pendencias.EmpresaIDs != nil {
		t.Errorf("administrador filtrado por empresas %v", repo.pendencias.EmpresaIDs)
	}

	if _, err := usecase.Prazos(c.ctx, portsin.ConsultarPrazosInput{UsuarioID: c.vendedor}); err != nil {
		t.Fatalf("Prazos falhou: %v", err)
	}
	if repo.prazos.RepresentanteID == nil || *repo.prazos.RepresentanteID != c.representante.ID {
		t.Errorf("vendedor consultou prazos com representante %v", repo.prazos.RepresentanteID)
	}

	for nome, consultar := range map[string]func() error{
		"pendências": func() error {
			_, err := usecase.Pendencias(c.ctx, portsin.ConsultarPendenciasInput{UsuarioID: c.comercial, Empresa: &c.red})
			return err
		},
		"prazos": func() error {
			_, err := usecase.Prazos(c.ctx, portsin.ConsultarPrazosInput{UsuarioID: c.comercial, Empresa: &c.red})
			return err
		},
		"cancelamentos": func() error {
			_, err := usecase.Cancelamentos(c.ctx, portsin.ConsultarCancelamentosInput{UsuarioID: c.comercial, Empresa: &c.red})
			return err
		},
	} {
		if err := consultar(); !errors.Is(err, domain.ErrPermissao) {
			t.Errorf("%s de empresa sem acesso = %v, esperado permissão", nome, err)
		}
	}
	if _, err := usecase.Pendencias(c.ctx, portsin.ConsultarPendenciasInput{UsuarioID: uuid.New()}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("usuário inexistente = %v, esperado não encontrado", err)
	}
}

func TestPrazosMontaAJanelaPelaSituacao(t *testing.T) {
	c := novoCenario(t)
	usecase, repo := c.consultas()
	hoje := domainpedido.Hoje(time.Now())
	sete := 7

	if _, err := usecase.Prazos(c.ctx, portsin.ConsultarPrazosInput{UsuarioID: c.comercial, Situacao: "a_vencer", Dias: &sete}); err != nil {
		t.Fatalf("Prazos falhou: %v", err)
	}
	janela := repo.prazos.Janela
	if janela.De == nil || !janela.De.Equal(hoje) || !janela.Ate.Equal(hoje.AddDate(0, 0, 7)) {
		t.Errorf("janela a vencer = %+v", janela)
	}

	if _, err := usecase.Prazos(c.ctx, portsin.ConsultarPrazosInput{UsuarioID: c.comercial, Situacao: "vencido"}); err != nil {
		t.Fatalf("Prazos falhou: %v", err)
	}
	if repo.prazos.Janela.De != nil || !repo.prazos.Janela.Ate.Equal(hoje.AddDate(0, 0, -1)) {
		t.Errorf("janela de vencidos = %+v", repo.prazos.Janela)
	}

	if _, err := usecase.Prazos(c.ctx, portsin.ConsultarPrazosInput{UsuarioID: c.comercial, Situacao: "atrasado"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("situação inválida = %v, esperado erro de validação", err)
	}
	muitos := 400
	if _, err := usecase.Prazos(c.ctx, portsin.ConsultarPrazosInput{UsuarioID: c.comercial, Dias: &muitos}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("dias acima do limite = %v, esperado erro de validação", err)
	}
}

func TestCancelamentosValidaOPeriodo(t *testing.T) {
	c := novoCenario(t)
	usecase, repo := c.consultas()
	hoje := time.Now().UTC()
	ontem := hoje.AddDate(0, 0, -1)

	if _, err := usecase.Cancelamentos(c.ctx, portsin.ConsultarCancelamentosInput{UsuarioID: c.comercial, De: &hoje, Ate: &ontem}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("período invertido = %v, esperado erro de validação", err)
	}

	if _, err := usecase.Cancelamentos(c.ctx, portsin.ConsultarCancelamentosInput{UsuarioID: c.comercial, ComDevolucao: true, De: &ontem, Ate: &hoje}); err != nil {
		t.Fatalf("Cancelamentos falhou: %v", err)
	}
	if !repo.cancelamentos.ComDevolucao || repo.cancelamentos.De == nil || repo.cancelamentos.Ate == nil || len(repo.cancelamentos.EmpresaIDs) != 2 {
		t.Errorf("filtro de cancelamentos = %+v", repo.cancelamentos)
	}
}
