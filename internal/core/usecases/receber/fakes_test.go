package receber_test

import (
	"context"
	"slices"
	"sort"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

type memoria struct {
	usuarios     map[uuid.UUID]*domainusuarios.Usuario
	cargos       map[uuid.UUID]*domainusuarios.Cargo
	empresas     map[uuid.UUID]*domainempresa.Empresa
	acessos      map[uuid.UUID]domainempresa.Acesso
	clientes     map[uuid.UUID]*domaincliente.Cliente
	contas       map[uuid.UUID]*domainfinanceiro.ContaBancaria
	categorias   map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
	pedidos      map[uuid.UUID]*domainpedido.Pedido
	geracoes     []*domainreceber.Geracao
	baixas       []*portsout.Baixa
	fornecedores map[uuid.UUID]*domainfornecedor.Fornecedor
	antecipacoes map[uuid.UUID]*domainreceber.Antecipacao
	recompras    []*portsout.GravacaoRecompra
	estornos     []*portsout.EstornoDaAntecipacao
	recebimentos map[uuid.UUID][]*domainreceber.Recebimento
	titulos      map[uuid.UUID]*domainreceber.Titulo
	historicos   []*domainreceber.Historico
	alteracoes   []*portsout.Alteracao
	filtros      []portsout.TituloFiltro
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:     map[uuid.UUID]*domainusuarios.Usuario{},
		cargos:       map[uuid.UUID]*domainusuarios.Cargo{},
		empresas:     map[uuid.UUID]*domainempresa.Empresa{},
		acessos:      map[uuid.UUID]domainempresa.Acesso{},
		clientes:     map[uuid.UUID]*domaincliente.Cliente{},
		contas:       map[uuid.UUID]*domainfinanceiro.ContaBancaria{},
		categorias:   map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira{},
		pedidos:      map[uuid.UUID]*domainpedido.Pedido{},
		recebimentos: map[uuid.UUID][]*domainreceber.Recebimento{},
		fornecedores: map[uuid.UUID]*domainfornecedor.Fornecedor{},
		antecipacoes: map[uuid.UUID]*domainreceber.Antecipacao{},
		titulos:      map[uuid.UUID]*domainreceber.Titulo{},
	}
}

func copiar[T any](item *T) *T {
	if item == nil {
		return nil
	}
	copia := *item

	return &copia
}

type repoUsuarios struct{ *memoria }

func (r repoUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return copiar(r.usuarios[id]), nil
}

type repoCargos struct{ *memoria }

func (r repoCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return copiar(r.cargos[id]), nil
}

type repoEmpresas struct{ *memoria }

func (r repoEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	return copiar(r.empresas[id]), nil
}

func (r repoEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	return r.acessos[usuarioID], nil
}

type repoClientes struct{ *memoria }

func (r repoClientes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Cliente, error) {
	return copiar(r.clientes[id]), nil
}

type repoContas struct{ *memoria }

func (r repoContas) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	return copiar(r.contas[id]), nil
}

type repoCategorias struct{ *memoria }

func (r repoCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	return copiar(r.categorias[id]), nil
}

type repoTitulos struct{ *memoria }

func (r repoTitulos) Criar(_ context.Context, titulo *domainreceber.Titulo, historico *domainreceber.Historico) error {
	r.titulos[titulo.ID] = copiar(titulo)
	r.historicos = append(r.historicos, historico)

	return nil
}

func (r repoTitulos) Atualizar(_ context.Context, alteracao *portsout.Alteracao) error {
	atual, ok := r.titulos[alteracao.Titulo.ID]
	if !ok {
		return domainreceber.ErroTituloNaoEncontrado()
	}
	if atual.Versao != alteracao.VersaoLida || atual.Status != alteracao.StatusAnterior {
		return domainreceber.ErroVersaoDivergente()
	}

	alteracao.Titulo.Versao = atual.Versao + 1
	r.titulos[alteracao.Titulo.ID] = copiar(alteracao.Titulo)
	r.alteracoes = append(r.alteracoes, alteracao)
	if alteracao.Historico != nil {
		r.historicos = append(r.historicos, alteracao.Historico)
	}

	return nil
}

func (r repoTitulos) Obter(_ context.Context, id uuid.UUID) (*domainreceber.Titulo, error) {
	return copiar(r.titulos[id]), nil
}

func (r repoTitulos) Listar(_ context.Context, filtro portsout.TituloFiltro) ([]*domainreceber.Titulo, error) {
	r.memoria.filtros = append(r.memoria.filtros, filtro)

	titulos := make([]*domainreceber.Titulo, 0)
	for _, titulo := range r.titulos {
		if filtro.Empresa != nil && titulo.EmpresaID != *filtro.Empresa {
			continue
		}
		if filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, titulo.EmpresaID) {
			continue
		}
		if filtro.Status != "" && titulo.Status != filtro.Status {
			continue
		}
		if filtro.Origem != "" && titulo.Origem != filtro.Origem {
			continue
		}
		if filtro.OrigemID != nil && (titulo.OrigemID == nil || *titulo.OrigemID != *filtro.OrigemID) {
			continue
		}
		if filtro.VencidosEm != nil && !titulo.Vencido(*filtro.VencidosEm) {
			continue
		}
		titulos = append(titulos, copiar(titulo))
	}
	sort.Slice(titulos, func(i, j int) bool { return titulos[i].Vencimento.Before(titulos[j].Vencimento) })

	return titulos, nil
}

func (r repoTitulos) ListarHistorico(_ context.Context, tituloID uuid.UUID) ([]*domainreceber.Historico, error) {
	historicos := make([]*domainreceber.Historico, 0)
	for _, historico := range r.historicos {
		if historico.TituloID == tituloID {
			historicos = append(historicos, historico)
		}
	}

	return historicos, nil
}

func (r repoTitulos) GerarDeOrigem(_ context.Context, geracao *domainreceber.Geracao) (domainreceber.ResultadoGeracao, error) {
	r.memoria.geracoes = append(r.memoria.geracoes, geracao)

	resultado := domainreceber.ResultadoGeracao{}
	for indice, titulo := range geracao.Titulos {
		existente := r.porOrigem(titulo.Origem, *titulo.OrigemID)
		if existente != nil {
			resultado.JaExistentes = append(resultado.JaExistentes, existente.ID)
			continue
		}
		r.titulos[titulo.ID] = copiar(titulo)
		r.historicos = append(r.historicos, geracao.Historicos[indice])
		resultado.Gerados = append(resultado.Gerados, titulo.ID)
	}

	return resultado, nil
}

func (r repoTitulos) CancelarPorOrigem(_ context.Context, cancelamento domainreceber.CancelamentoDeOrigem) (domainreceber.ResultadoCancelamento, error) {
	resultado := domainreceber.ResultadoCancelamento{}
	for _, origemID := range cancelamento.OrigemIDs {
		titulo := r.porOrigem(cancelamento.Origem, origemID)
		if titulo == nil {
			resultado.Ignorados++
			continue
		}
		switch efeito, _ := titulo.AplicarCancelamentoDaOrigem(cancelamento); efeito {
		case domainreceber.EfeitoCancelado:
			resultado.Cancelados = append(resultado.Cancelados, titulo.ID)
		case domainreceber.EfeitoMarcado:
			resultado.Marcados = append(resultado.Marcados, titulo.ID)
		default:
			resultado.Ignorados++
		}
	}

	return resultado, nil
}

func (r repoTitulos) porOrigem(origem domainreceber.Origem, origemID uuid.UUID) *domainreceber.Titulo {
	for _, titulo := range r.titulos {
		if titulo.Origem == origem && titulo.OrigemID != nil && *titulo.OrigemID == origemID {
			return titulo
		}
	}

	return nil
}

type repoPedidos struct{ *memoria }

func (r repoPedidos) Obter(_ context.Context, id uuid.UUID) (*domainpedido.Pedido, error) {
	return copiar(r.pedidos[id]), nil
}

func (r repoTitulos) Receber(ctx context.Context, baixa *portsout.Baixa) error {
	if err := r.Atualizar(ctx, &portsout.Alteracao{
		Titulo: baixa.Titulo, VersaoLida: baixa.VersaoLida, StatusAnterior: baixa.StatusAnterior, Historico: baixa.Historico,
	}); err != nil {
		return err
	}
	r.memoria.baixas = append(r.memoria.baixas, baixa)
	r.recebimentos[baixa.Titulo.ID] = append(r.recebimentos[baixa.Titulo.ID], copiar(baixa.Recebimento))
	if baixa.Entrada != nil {
		baixa.Entrada.Pedido.Versao++
		r.pedidos[baixa.Entrada.Pedido.ID] = copiar(baixa.Entrada.Pedido)
	}

	return nil
}

func (r repoTitulos) EstornarRecebimento(ctx context.Context, estorno *portsout.EstornoDaBaixa) error {
	for _, recebimento := range r.recebimentos[estorno.Titulo.ID] {
		if recebimento.ID == estorno.Recebimento.ID {
			recebimento.Estornado = true
			recebimento.Estorno = estorno.Recebimento.Estorno
		}
	}

	return r.Atualizar(ctx, &portsout.Alteracao{
		Titulo: estorno.Titulo, VersaoLida: estorno.VersaoLida, StatusAnterior: estorno.StatusAnterior, Historico: estorno.Historico,
	})
}

func (r repoTitulos) ListarRecebimentos(_ context.Context, tituloID uuid.UUID) ([]*domainreceber.Recebimento, error) {
	recebimentos := make([]*domainreceber.Recebimento, 0, len(r.recebimentos[tituloID]))
	for _, recebimento := range r.recebimentos[tituloID] {
		recebimentos = append(recebimentos, copiar(recebimento))
	}

	return recebimentos, nil
}

type repoFornecedores struct{ *memoria }

func (r repoFornecedores) Obter(_ context.Context, id uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	return copiar(r.fornecedores[id]), nil
}

type repoAntecipacoes struct{ *memoria }

func copiarAntecipacao(antecipacao *domainreceber.Antecipacao) *domainreceber.Antecipacao {
	if antecipacao == nil {
		return nil
	}
	copia := *antecipacao
	copia.Titulos = make([]*domainreceber.AntecipacaoTitulo, 0, len(antecipacao.Titulos))
	for _, vinculo := range antecipacao.Titulos {
		copia.Titulos = append(copia.Titulos, copiar(vinculo))
	}

	return &copia
}

func (r repoAntecipacoes) gravarTitulos(ctx context.Context, titulos []portsout.TituloAlterado) error {
	for _, alterado := range titulos {
		if err := repoTitulos(r).Atualizar(ctx, &portsout.Alteracao{
			Titulo: alterado.Titulo, VersaoLida: alterado.VersaoLida, StatusAnterior: alterado.StatusAnterior, Historico: alterado.Historico,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r repoAntecipacoes) Criar(ctx context.Context, gravacao *portsout.GravacaoAntecipacao) error {
	if err := r.gravarTitulos(ctx, gravacao.Titulos); err != nil {
		return err
	}
	r.antecipacoes[gravacao.Antecipacao.ID] = copiarAntecipacao(gravacao.Antecipacao)

	return nil
}

func (r repoAntecipacoes) Obter(_ context.Context, id uuid.UUID) (*domainreceber.Antecipacao, error) {
	return copiarAntecipacao(r.antecipacoes[id]), nil
}

func (r repoAntecipacoes) ObterVigenteDoTitulo(_ context.Context, tituloID uuid.UUID) (*domainreceber.Antecipacao, error) {
	for _, antecipacao := range r.antecipacoes {
		if vinculo := antecipacao.Vinculo(tituloID); vinculo != nil && vinculo.Vigente {
			return copiarAntecipacao(antecipacao), nil
		}
	}

	return nil, nil
}

func (r repoAntecipacoes) Listar(_ context.Context, filtro portsout.AntecipacaoFiltro) ([]*domainreceber.Antecipacao, error) {
	antecipacoes := make([]*domainreceber.Antecipacao, 0)
	for _, antecipacao := range r.antecipacoes {
		if filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, antecipacao.EmpresaID) {
			continue
		}
		if filtro.Status != "" && antecipacao.Status != filtro.Status {
			continue
		}
		antecipacoes = append(antecipacoes, copiarAntecipacao(antecipacao))
	}

	return antecipacoes, nil
}

func (r repoAntecipacoes) Estornar(ctx context.Context, estorno *portsout.EstornoDaAntecipacao) error {
	if r.antecipacoes[estorno.Antecipacao.ID].Versao != estorno.VersaoLida {
		return domainreceber.ErroAntecipacaoAlterada()
	}
	if err := r.gravarTitulos(ctx, estorno.Titulos); err != nil {
		return err
	}
	estorno.Antecipacao.Versao++
	r.antecipacoes[estorno.Antecipacao.ID] = copiarAntecipacao(estorno.Antecipacao)
	r.memoria.estornos = append(r.memoria.estornos, estorno)

	return nil
}

func (r repoAntecipacoes) Recomprar(ctx context.Context, gravacao *portsout.GravacaoRecompra) error {
	if err := r.gravarTitulos(ctx, []portsout.TituloAlterado{gravacao.Titulo}); err != nil {
		return err
	}
	r.antecipacoes[gravacao.Recompra.AntecipacaoID].Vinculo(gravacao.Recompra.TituloID).Vigente = false
	gravacao.Recompra.SolicitacaoID = gravacao.ContaAPagar.Solicitacao.ID
	r.memoria.recompras = append(r.memoria.recompras, gravacao)

	return nil
}
