package dre

import (
	"context"
	"time"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
)

var _ portsin.RelatorioSemanalUseCase = (*RelatorioSemanalUsecaseImpl)(nil)

type RelatorioSemanalUsecaseImpl struct {
	relatorios portsout.RelatorioSemanalRepository
	saldos     portsout.SaldoRepository
	dividas    portsout.DividaRepository
	permissoes
}

func NewRelatorioSemanalUsecase(
	relatorios portsout.RelatorioSemanalRepository,
	saldos portsout.SaldoRepository,
	dividas portsout.DividaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *RelatorioSemanalUsecaseImpl {
	return &RelatorioSemanalUsecaseImpl{
		relatorios: relatorios,
		saldos:     saldos,
		dividas:    dividas,
		permissoes: permissoes{usuarios: usuarios, cargos: cargos, empresas: empresas},
	}
}

func (u *RelatorioSemanalUsecaseImpl) Consultar(ctx context.Context, input portsin.ConsultarSemanaInput) (domaindre.RelatorioSemanal, error) {
	escopo, err := u.escopoDoUsuario(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return domaindre.RelatorioSemanal{}, err
	}

	hoje := domaindre.Hoje(time.Now())
	semana, err := domaindre.NovaSemana(input.Semana, hoje)
	if err != nil {
		return domaindre.RelatorioSemanal{}, err
	}

	filtro := portsout.FiltroSemanal{
		Intervalo:  semana.Intervalo(),
		Hoje:       hoje,
		EmpresaIDs: escopo.empresaIDs,
		Limite:     domaindre.LimiteDasListasDoRelatorio,
	}

	var conteudo domaindre.ConteudoDoRelatorio
	if conteudo.Notas, err = u.relatorios.NotasEmitidas(ctx, filtro); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}
	if conteudo.Pagamentos, err = u.relatorios.PagamentosFeitos(ctx, filtro); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}
	if conteudo.Recebimentos, err = u.relatorios.RecebimentosFeitos(ctx, filtro); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}
	if conteudo.TitulosVencidos, err = u.relatorios.TitulosVencidos(ctx, filtro); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}
	if conteudo.ContasVencidas, err = u.relatorios.ContasVencidas(ctx, filtro); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}
	if conteudo.EstoqueBaixo, err = u.relatorios.EstoqueBaixo(ctx, filtro.Limite); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}
	if conteudo.Saldos, err = u.saldos.Saldos(ctx, escopo.empresaIDs); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}
	if conteudo.Parcelas, err = u.dividas.ParcelasEmAberto(ctx, escopo.empresaIDs); err != nil {
		return domaindre.RelatorioSemanal{}, err
	}

	return domaindre.NovoRelatorioSemanal(semana, hoje, conteudo), nil
}
