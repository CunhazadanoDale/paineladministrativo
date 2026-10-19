package caixa_test

import (
	"bytes"
	"context"
	"io"
	"slices"
	"time"

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
	contas        *repositorioContas
	mesesFechados []time.Time
	importacoes   []*domaincaixa.ImportacaoExtrato
	lancamentos   []*domaincaixa.LancamentoExtrato
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

func (r *repositorioExtratos) MesesFechados(_ context.Context, _ uuid.UUID) ([]time.Time, error) {
	return r.mesesFechados, nil
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

var _ portsout.ConciliacaoRepository = (*repositorioConciliacoes)(nil)

type repositorioConciliacoes struct {
	lancamentos  map[uuid.UUID]*domaincaixa.LancamentoExtrato
	movimentos   []*domaincaixa.MovimentoSistema
	semConta     []*domaincaixa.MovimentoSistema
	conciliacoes map[uuid.UUID]*domaincaixa.Conciliacao
	filtros      []portsout.MovimentoFiltro
	falharCriar  error
}

func novoRepositorioConciliacoes() *repositorioConciliacoes {
	return &repositorioConciliacoes{
		lancamentos:  map[uuid.UUID]*domaincaixa.LancamentoExtrato{},
		conciliacoes: map[uuid.UUID]*domaincaixa.Conciliacao{},
	}
}

func (r *repositorioConciliacoes) ObterLancamento(_ context.Context, id uuid.UUID) (*domaincaixa.LancamentoExtrato, error) {
	lancamento, ok := r.lancamentos[id]
	if !ok {
		return nil, nil
	}

	copia := *lancamento
	return &copia, nil
}

func (r *repositorioConciliacoes) LancamentosPendentes(_ context.Context, contaBancariaID uuid.UUID, de, ate time.Time, limite int) ([]*domaincaixa.LancamentoExtrato, error) {
	var lancamentos []*domaincaixa.LancamentoExtrato
	for _, lancamento := range r.lancamentos {
		if lancamento.ContaBancariaID != contaBancariaID || lancamento.ConciliacaoID != nil ||
			lancamento.Data.Before(de) || lancamento.Data.After(ate) {
			continue
		}
		copia := *lancamento
		lancamentos = append(lancamentos, &copia)
	}
	slices.SortFunc(lancamentos, func(a, b *domaincaixa.LancamentoExtrato) int { return a.Data.Compare(b.Data) })
	if len(lancamentos) > limite {
		lancamentos = lancamentos[:limite]
	}

	return lancamentos, nil
}

func (r *repositorioConciliacoes) Movimentos(_ context.Context, filtro portsout.MovimentoFiltro) ([]*domaincaixa.MovimentoSistema, error) {
	r.filtros = append(r.filtros, filtro)

	var movimentos []*domaincaixa.MovimentoSistema
	for _, movimento := range r.movimentos {
		if movimento.ContaBancariaID != filtro.ContaBancariaID ||
			(filtro.SoPendentes && !movimento.Pendente()) ||
			(filtro.ValorCentavos != nil && movimento.ValorCentavos != *filtro.ValorCentavos) ||
			(filtro.De != nil && movimento.Data.Before(*filtro.De)) ||
			(filtro.Ate != nil && movimento.Data.After(*filtro.Ate)) {
			continue
		}
		copia := *movimento
		movimentos = append(movimentos, &copia)
	}

	return movimentos, nil
}

func (r *repositorioConciliacoes) MovimentosPorReferencia(_ context.Context, contaBancariaID uuid.UUID, referencias []domaincaixa.ReferenciaMovimento) ([]*domaincaixa.MovimentoSistema, error) {
	var movimentos []*domaincaixa.MovimentoSistema
	for _, movimento := range r.movimentos {
		if movimento.ContaBancariaID == contaBancariaID && slices.Contains(referencias, movimento.Referencia()) {
			copia := *movimento
			movimentos = append(movimentos, &copia)
		}
	}

	return movimentos, nil
}

func (r *repositorioConciliacoes) PagamentosSemConta(_ context.Context, filtro portsout.SemContaFiltro) ([]*domaincaixa.MovimentoSistema, error) {
	var movimentos []*domaincaixa.MovimentoSistema
	for _, movimento := range r.semConta {
		if filtro.EmpresaIDs == nil || slices.Contains(filtro.EmpresaIDs, movimento.EmpresaID) {
			movimentos = append(movimentos, movimento)
		}
	}

	return movimentos, nil
}

func (r *repositorioConciliacoes) Criar(_ context.Context, conciliacao *domaincaixa.Conciliacao) error {
	if r.falharCriar != nil {
		return r.falharCriar
	}

	copia := *conciliacao
	copia.Itens = slices.Clone(conciliacao.Itens)
	r.conciliacoes[conciliacao.ID] = &copia
	r.lancamentos[conciliacao.ExtratoLancamentoID].ConciliacaoID = &copia.ID
	for _, item := range conciliacao.Itens {
		for _, movimento := range r.movimentos {
			if movimento.Tipo == item.Tipo && movimento.ID == item.AlvoID {
				movimento.ConciliacaoID = &copia.ID
			}
		}
	}

	return nil
}

func (r *repositorioConciliacoes) Obter(_ context.Context, id uuid.UUID) (*domaincaixa.Conciliacao, error) {
	conciliacao, ok := r.conciliacoes[id]
	if !ok {
		return nil, nil
	}

	copia := *conciliacao
	copia.Itens = slices.Clone(conciliacao.Itens)
	return &copia, nil
}

func (r *repositorioConciliacoes) Desfazer(_ context.Context, conciliacao *domaincaixa.Conciliacao) error {
	atual := r.conciliacoes[conciliacao.ID]
	if !atual.Vigente {
		return domaincaixa.ErroConciliacaoJaDesfeita()
	}

	copia := *conciliacao
	r.conciliacoes[conciliacao.ID] = &copia
	r.lancamentos[conciliacao.ExtratoLancamentoID].ConciliacaoID = nil
	for _, movimento := range r.movimentos {
		if movimento.ConciliacaoID != nil && *movimento.ConciliacaoID == conciliacao.ID {
			movimento.ConciliacaoID = nil
		}
	}

	return nil
}

var (
	_ portsout.LancamentoCaixaRepository     = (*repositorioLancamentosCaixa)(nil)
	_ portsout.CategoriaFinanceiraRepository = (*repositorioCategorias)(nil)
)

type repositorioCategorias struct {
	itens map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
}

func (r *repositorioCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	categoria, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *categoria
	return &copia, nil
}

type repositorioLancamentosCaixa struct {
	classificacoes []*domaincaixa.Classificacao
	transferencias []*domaincaixa.TransferenciaRegistrada
	filtros        []portsout.LancamentoCaixaFiltro
}

func (r *repositorioLancamentosCaixa) Classificar(_ context.Context, classificacao *domaincaixa.Classificacao) error {
	r.classificacoes = append(r.classificacoes, classificacao)
	return nil
}

func (r *repositorioLancamentosCaixa) Transferir(_ context.Context, registrada *domaincaixa.TransferenciaRegistrada) error {
	r.transferencias = append(r.transferencias, registrada)
	return nil
}

func (r *repositorioLancamentosCaixa) Listar(_ context.Context, filtro portsout.LancamentoCaixaFiltro) ([]*domaincaixa.LancamentoCaixa, error) {
	r.filtros = append(r.filtros, filtro)

	var lancamentos []*domaincaixa.LancamentoCaixa
	for _, classificacao := range r.classificacoes {
		lancamentos = append(lancamentos, classificacao.Lancamento)
	}

	return lancamentos, nil
}

func (r *repositorioLancamentosCaixa) ListarTransferencias(_ context.Context, filtro portsout.LancamentoCaixaFiltro) ([]*domaincaixa.Transferencia, error) {
	r.filtros = append(r.filtros, filtro)

	var transferencias []*domaincaixa.Transferencia
	for _, registrada := range r.transferencias {
		transferencias = append(transferencias, registrada.Transferencia)
	}

	return transferencias, nil
}

var _ portsout.FechamentoRepository = (*repositorioFechamentos)(nil)

type repositorioFechamentos struct {
	itens     []*domaincaixa.Fechamento
	movimento domaincaixa.MovimentoDoMes
	gravacoes []*portsout.GravacaoFechamento
	filtros   []portsout.FechamentoFiltro
}

func (r *repositorioFechamentos) Ultimo(_ context.Context, contaBancariaID uuid.UUID) (*domaincaixa.Fechamento, error) {
	var ultimo *domaincaixa.Fechamento
	for _, fechamento := range r.itens {
		if fechamento.ContaBancariaID == contaBancariaID && fechamento.Vigente &&
			(ultimo == nil || fechamento.Competencia.After(ultimo.Competencia)) {
			ultimo = fechamento
		}
	}
	if ultimo == nil {
		return nil, nil
	}

	copia := *ultimo
	return &copia, nil
}

func (r *repositorioFechamentos) MovimentoDoMes(_ context.Context, _ uuid.UUID, _, _ time.Time) (*domaincaixa.MovimentoDoMes, error) {
	movimento := r.movimento
	return &movimento, nil
}

func (r *repositorioFechamentos) Fechar(_ context.Context, gravacao *portsout.GravacaoFechamento) error {
	r.gravacoes = append(r.gravacoes, gravacao)
	copia := *gravacao.Fechamento
	r.itens = append(r.itens, &copia)
	return nil
}

func (r *repositorioFechamentos) Obter(_ context.Context, id uuid.UUID) (*domaincaixa.Fechamento, error) {
	for _, fechamento := range r.itens {
		if fechamento.ID == id {
			copia := *fechamento
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioFechamentos) Listar(_ context.Context, filtro portsout.FechamentoFiltro) ([]*domaincaixa.Fechamento, error) {
	r.filtros = append(r.filtros, filtro)
	return r.itens, nil
}

func (r *repositorioFechamentos) Reabrir(_ context.Context, fechamento *domaincaixa.Fechamento) error {
	for i, item := range r.itens {
		if item.ID == fechamento.ID {
			copia := *fechamento
			r.itens[i] = &copia
		}
	}

	return nil
}

var (
	_ portsout.ProjecaoRepository          = (*repositorioProjecao)(nil)
	_ portsout.DespesaRecorrenteRepository = (*repositorioRecorrencias)(nil)
)

type repositorioProjecao struct {
	partida      domaincaixa.SaldoDePartida
	entradas     []domaincaixa.Previsto
	saidas       []domaincaixa.Previsto
	geradas      map[portsout.RecorrenciaNaCompetencia]bool
	consultadas  []portsout.RecorrenciaNaCompetencia
	empresas     [][]uuid.UUID
	movimentos   []*domaincaixa.MovimentoDiario
	filtroDiario portsout.MovimentoDiarioFiltro
}

func (r *repositorioProjecao) SaldoDePartida(_ context.Context, empresaIDs []uuid.UUID) (*domaincaixa.SaldoDePartida, error) {
	r.empresas = append(r.empresas, empresaIDs)
	partida := r.partida
	return &partida, nil
}

func (r *repositorioProjecao) TitulosAReceber(_ context.Context, _ []uuid.UUID, _ time.Time) ([]domaincaixa.Previsto, error) {
	return r.entradas, nil
}

func (r *repositorioProjecao) ContasAPagar(_ context.Context, _ []uuid.UUID, _ time.Time) ([]domaincaixa.Previsto, error) {
	return r.saidas, nil
}

func (r *repositorioProjecao) RecorrenciasGeradas(_ context.Context, candidatas []portsout.RecorrenciaNaCompetencia) (map[portsout.RecorrenciaNaCompetencia]bool, error) {
	r.consultadas = append(r.consultadas, candidatas...)
	return r.geradas, nil
}

func (r *repositorioProjecao) MovimentoDiario(_ context.Context, filtro portsout.MovimentoDiarioFiltro) ([]*domaincaixa.MovimentoDiario, error) {
	r.filtroDiario = filtro
	return r.movimentos, nil
}

type repositorioRecorrencias struct {
	itens []*domainsolicitacao.DespesaRecorrente
}

func (r *repositorioRecorrencias) ListarVigentes(_ context.Context, competencia domainsolicitacao.Competencia, empresaIDs []uuid.UUID) ([]*domainsolicitacao.DespesaRecorrente, error) {
	var vigentes []*domainsolicitacao.DespesaRecorrente
	for _, despesa := range r.itens {
		if despesa.VigenteEm(competencia) && (empresaIDs == nil || slices.Contains(empresaIDs, despesa.EmpresaID)) {
			vigentes = append(vigentes, despesa)
		}
	}

	return vigentes, nil
}
