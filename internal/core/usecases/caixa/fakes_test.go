package caixa_test

import (
	"bytes"
	"context"
	"io"
	"slices"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

var (
	_ portsout.EmpresaRepository       = (*repositorioEmpresas)(nil)
	_ portsout.UsuarioRepository       = (*repositorioUsuarios)(nil)
	_ portsout.CargoRepository         = (*repositorioCargos)(nil)
	_ portsout.ContaBancariaRepository = (*repositorioContas)(nil)
	_ portsout.ArquivoRepository       = (*repositorioArquivos)(nil)
	_ portsout.Storage                 = (*armazenamento)(nil)
	_ portsout.AberturaRepository      = (*repositorioAberturas)(nil)
	_ portsout.ExtratoRepository       = (*repositorioExtratos)(nil)
	_ portsout.SaldoRepository         = (*repositorioSaldos)(nil)
)

type repositorioEmpresas struct {
	itens   map[uuid.UUID]*domainempresa.Empresa
	acessos map[uuid.UUID]domainempresa.Acesso
}

func (r *repositorioEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	empresa, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *empresa
	return &copia, nil
}

func (r *repositorioEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	if acesso, ok := r.acessos[usuarioID]; ok {
		return acesso, nil
	}

	return domainempresa.AcessoRestrito(nil), nil
}

type repositorioUsuarios struct {
	itens map[uuid.UUID]*domainusuarios.Usuario
}

func (r *repositorioUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	usuario, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *usuario
	return &copia, nil
}

type repositorioCargos struct {
	itens map[uuid.UUID]*domainusuarios.Cargo
}

func (r *repositorioCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	cargo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *cargo
	return &copia, nil
}

type repositorioContas struct {
	itens map[uuid.UUID]*domainfinanceiro.ContaBancaria
}

func (r *repositorioContas) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	conta, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *conta
	return &copia, nil
}

type repositorioArquivos struct {
	itens         map[uuid.UUID]*domainsolicitacao.Arquivo
	daSolicitacao map[uuid.UUID]bool
}

func (r *repositorioArquivos) Obter(_ context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error) {
	arquivo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *arquivo
	return &copia, nil
}

func (r *repositorioArquivos) VinculadoASolicitacao(_ context.Context, arquivoID uuid.UUID) (bool, error) {
	return r.daSolicitacao[arquivoID], nil
}

type armazenamento struct {
	conteudos map[string][]byte
}

func (a *armazenamento) Baixar(_ context.Context, chave string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(a.conteudos[chave])), nil
}

type repositorioAberturas struct {
	itens map[uuid.UUID]*domaincaixa.Abertura
}

func (r *repositorioAberturas) Obter(_ context.Context, contaBancariaID uuid.UUID) (*domaincaixa.Abertura, error) {
	abertura, ok := r.itens[contaBancariaID]
	if !ok {
		return nil, nil
	}

	copia := *abertura
	return &copia, nil
}

func (r *repositorioAberturas) Criar(_ context.Context, abertura *domaincaixa.Abertura) error {
	if _, ok := r.itens[abertura.ContaBancariaID]; ok {
		return domaincaixa.ErroAberturaJaDefinida()
	}

	copia := *abertura
	r.itens[abertura.ContaBancariaID] = &copia
	return nil
}

func (r *repositorioAberturas) Atualizar(_ context.Context, abertura *domaincaixa.Abertura) error {
	atual, ok := r.itens[abertura.ContaBancariaID]
	if !ok || atual.Versao != abertura.Versao {
		return domaincaixa.ErroAberturaAlterada()
	}

	abertura.Versao++
	copia := *abertura
	r.itens[abertura.ContaBancariaID] = &copia
	return nil
}

type repositorioExtratos struct {
	contas      *repositorioContas
	importacoes []*domaincaixa.ImportacaoExtrato
	lancamentos []*domaincaixa.LancamentoExtrato
}

func (r *repositorioExtratos) ContasDoBanco(_ context.Context, banco domainfinanceiro.Banco) ([]*domainfinanceiro.ContaBancaria, error) {
	var contas []*domainfinanceiro.ContaBancaria
	for _, conta := range r.contas.itens {
		if conta.Banco == banco {
			copia := *conta
			contas = append(contas, &copia)
		}
	}

	return contas, nil
}

func (r *repositorioExtratos) ArquivoImportado(_ context.Context, arquivoID uuid.UUID) (bool, error) {
	for _, importacao := range r.importacoes {
		if importacao.ArquivoID == arquivoID {
			return true, nil
		}
	}

	return false, nil
}

func (r *repositorioExtratos) ContarExistentes(_ context.Context, contaBancariaID uuid.UUID, identificadores []string) (int, error) {
	existentes := 0
	for _, lancamento := range r.lancamentos {
		if lancamento.ContaBancariaID == contaBancariaID && slices.Contains(identificadores, lancamento.Identificador) {
			existentes++
		}
	}

	return existentes, nil
}

func (r *repositorioExtratos) Importar(ctx context.Context, preparado *domaincaixa.ExtratoPreparado, _ int) error {
	novas := 0
	for _, lancamento := range preparado.Lancamentos {
		existentes, _ := r.ContarExistentes(ctx, lancamento.ContaBancariaID, []string{lancamento.Identificador})
		if existentes > 0 {
			continue
		}
		copia := *lancamento
		r.lancamentos = append(r.lancamentos, &copia)
		novas++
	}

	preparado.RegistrarNovas(novas)
	copia := *preparado.Importacao
	r.importacoes = append(r.importacoes, &copia)
	return nil
}

func (r *repositorioExtratos) ObterImportacao(_ context.Context, id uuid.UUID) (*domaincaixa.ImportacaoExtrato, error) {
	for _, importacao := range r.importacoes {
		if importacao.ID == id {
			copia := *importacao
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioExtratos) ListarImportacoes(_ context.Context, filtro portsout.ImportacaoFiltro) ([]*domaincaixa.ImportacaoExtrato, error) {
	var importacoes []*domaincaixa.ImportacaoExtrato
	for _, importacao := range r.importacoes {
		if filtro.ContaBancariaID != nil && importacao.ContaBancariaID != *filtro.ContaBancariaID {
			continue
		}
		if filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, r.contas.itens[importacao.ContaBancariaID].EmpresaID) {
			continue
		}
		copia := *importacao
		importacoes = append(importacoes, &copia)
	}

	return importacoes, nil
}

func (r *repositorioExtratos) DesfazerImportacao(_ context.Context, importacao *domaincaixa.ImportacaoExtrato) error {
	r.importacoes = slices.DeleteFunc(r.importacoes, func(item *domaincaixa.ImportacaoExtrato) bool {
		return item.ID == importacao.ID
	})
	r.lancamentos = slices.DeleteFunc(r.lancamentos, func(item *domaincaixa.LancamentoExtrato) bool {
		return item.ImportacaoID == importacao.ID
	})

	return nil
}

func (r *repositorioExtratos) ListarLancamentos(_ context.Context, filtro portsout.LancamentoFiltro) ([]*domaincaixa.LancamentoExtrato, error) {
	var lancamentos []*domaincaixa.LancamentoExtrato
	for _, lancamento := range r.lancamentos {
		if lancamento.ContaBancariaID != filtro.ContaBancariaID {
			continue
		}
		if filtro.De != nil && lancamento.Data.Before(*filtro.De) {
			continue
		}
		if filtro.Ate != nil && lancamento.Data.After(*filtro.Ate) {
			continue
		}
		copia := *lancamento
		lancamentos = append(lancamentos, &copia)
	}

	return lancamentos, nil
}

type repositorioSaldos struct {
	contas  *repositorioContas
	filtros [][]uuid.UUID
}

func (r *repositorioSaldos) Saldos(_ context.Context, empresaIDs []uuid.UUID) ([]*domaincaixa.SaldoConta, error) {
	r.filtros = append(r.filtros, empresaIDs)

	var saldos []*domaincaixa.SaldoConta
	for _, conta := range r.contas.itens {
		if empresaIDs != nil && !slices.Contains(empresaIDs, conta.EmpresaID) {
			continue
		}
		saldos = append(saldos, &domaincaixa.SaldoConta{ContaBancariaID: conta.ID, EmpresaID: conta.EmpresaID})
	}

	return saldos, nil
}
