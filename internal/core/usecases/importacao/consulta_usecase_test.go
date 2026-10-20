package importacao_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	importacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/importacao"
	"github.com/google/uuid"
)

type repoConsultas struct {
	alocacoes []domainimportacao.AlocacaoAberta
	pedidos   []domainimportacao.PedidoSemImportacao
	empresas  map[uuid.UUID]uuid.UUID
	filtro    portsout.AlertasFiltro
}

func (r *repoConsultas) AlocacoesAbertas(_ context.Context, filtro portsout.AlertasFiltro) ([]domainimportacao.AlocacaoAberta, error) {
	r.filtro = filtro
	encontradas := make([]domainimportacao.AlocacaoAberta, 0)
	for _, alocacao := range r.alocacoes {
		if filtro.EmpresaIDs == nil || slices.Contains(filtro.EmpresaIDs, alocacao.EmpresaID) {
			encontradas = append(encontradas, alocacao)
		}
	}

	return encontradas, nil
}

func (r *repoConsultas) PedidosComFalta(_ context.Context, _ portsout.AlertasFiltro) ([]domainimportacao.PedidoSemImportacao, error) {
	return r.pedidos, nil
}

func (r *repoConsultas) EmpresaDoPedido(_ context.Context, pedidoID uuid.UUID) (*uuid.UUID, error) {
	empresa, ok := r.empresas[pedidoID]
	if !ok {
		return nil, nil
	}

	return &empresa, nil
}

func (r *repoConsultas) AtendimentosDoPedido(_ context.Context, _ uuid.UUID) ([]domainimportacao.AtendimentoDoPedido, error) {
	return []domainimportacao.AtendimentoDoPedido{{ImportacaoNumero: 3}}, nil
}

func TestAlertasRespeitamOAcessoEAFolga(t *testing.T) {
	c := novoCenario(t)
	chegada := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	prazo := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)
	consultas := &repoConsultas{alocacoes: []domainimportacao.AlocacaoAberta{
		{EmpresaID: c.mia, ChegadaPrevista: &chegada, PrazoLimite: &prazo},
		{EmpresaID: c.red, ChegadaPrevista: &chegada, PrazoLimite: &prazo},
	}}
	usecase := importacaousecases.NewConsultaUsecase(consultas, repoImportacoes{c.m}, repoEmpresas{c.m}, repoUsuarios{c.m}, repoCargos{c.m})

	alertas, err := usecase.Alertas(c.ctx, portsin.AlertasInput{UsuarioID: c.comercial})
	if err != nil || alertas.Folga != 10 || len(alertas.PrazoEmRisco) != 1 || consultas.filtro.EmpresaIDs == nil {
		t.Errorf("alertas do comercial = %+v (%v), filtro %+v", alertas, err, consultas.filtro)
	}
	folga := 0
	if alertas, err := usecase.Alertas(c.ctx, portsin.AlertasInput{UsuarioID: c.administrador, FolgaDias: &folga}); err != nil || len(alertas.PrazoEmRisco) != 0 {
		t.Errorf("alertas sem folga = %+v (%v)", alertas, err)
	}

	invalida := 91
	if _, err := usecase.Alertas(c.ctx, portsin.AlertasInput{UsuarioID: c.comercial, FolgaDias: &invalida}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("folga inválida: erro = %v", err)
	}
	if _, err := usecase.Alertas(c.ctx, portsin.AlertasInput{UsuarioID: c.comercial, Empresa: &c.red}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso: erro = %v", err)
	}
	if _, err := usecase.Alertas(c.ctx, portsin.AlertasInput{UsuarioID: c.operador}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("operador: erro = %v", err)
	}
}

func TestImportacoesDoPedido(t *testing.T) {
	c := novoCenario(t)
	daMia, daRed := uuid.New(), uuid.New()
	consultas := &repoConsultas{empresas: map[uuid.UUID]uuid.UUID{daMia: c.mia, daRed: c.red}}
	usecase := importacaousecases.NewConsultaUsecase(consultas, repoImportacoes{c.m}, repoEmpresas{c.m}, repoUsuarios{c.m}, repoCargos{c.m})

	if atendimentos, err := usecase.ImportacoesDoPedido(c.ctx, daMia, c.financeiro); err != nil || len(atendimentos) != 1 {
		t.Errorf("importações do pedido = %+v (%v)", atendimentos, err)
	}
	if _, err := usecase.ImportacoesDoPedido(c.ctx, daRed, c.financeiro); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("pedido de outra empresa: erro = %v", err)
	}
	if _, err := usecase.ImportacoesDoPedido(c.ctx, uuid.New(), c.financeiro); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("pedido inexistente: erro = %v", err)
	}
}
