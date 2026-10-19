package caixa

import (
	"context"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

var _ portsin.ExtratoUseCase = (*ExtratoUsecaseImpl)(nil)

type ExtratoUsecaseImpl struct {
	extratos  portsout.ExtratoRepository
	aberturas portsout.AberturaRepository
	leitorDeOFX
	permissoes
	escopoPorEmpresa
}

func NewExtratoUsecase(
	extratos portsout.ExtratoRepository,
	aberturas portsout.AberturaRepository,
	arquivos portsout.ArquivoRepository,
	storage portsout.Storage,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ExtratoUsecaseImpl {
	return &ExtratoUsecaseImpl{
		extratos:         extratos,
		aberturas:        aberturas,
		leitorDeOFX:      leitorDeOFX{arquivos: arquivos, storage: storage, extratos: extratos},
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas, contas: contas},
	}
}

func (u *ExtratoUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarExtratoInput) (*portsin.RelatorioImportacao, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	conteudo, err := u.leitorDeOFX.ler(ctx, input.ArquivoID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	extrato, err := domaincaixa.LerOFX(conteudo)
	if err != nil {
		return nil, err
	}

	conta, err := u.contaDoExtrato(ctx, extrato, input.ContaBancariaID, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := u.exigeContaAtiva(ctx, conta); err != nil {
		return nil, err
	}

	abertura, err := u.aberturas.Obter(ctx, conta.ID)
	if err != nil {
		return nil, err
	}
	if abertura == nil {
		return nil, domaincaixa.ErroContaSemAbertura()
	}

	mesesFechados, err := u.extratos.MesesFechados(ctx, conta.ID)
	if err != nil {
		return nil, err
	}

	preparado, err := domaincaixa.PrepararImportacao(domaincaixa.DadosImportacao{
		Extrato:         extrato,
		ContaBancariaID: conta.ID,
		ArquivoID:       input.ArquivoID,
		UsuarioID:       input.UsuarioID,
		Abertura:        abertura,
		MesesFechados:   mesesFechados,
		Agora:           time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	if input.Simular {
		existentes, err := u.extratos.ContarExistentes(ctx, conta.ID, preparado.Identificadores())
		if err != nil {
			return nil, err
		}
		preparado.RegistrarNovas(len(preparado.Lancamentos) - existentes)
	} else if err := u.extratos.Importar(ctx, preparado, abertura.Versao); err != nil {
		return nil, err
	}

	return &portsin.RelatorioImportacao{
		Simulacao:  input.Simular,
		Importacao: preparado.Importacao,
		Total:      preparado.Total(),
		Ignoradas:  preparado.Ignoradas,
	}, nil
}

func (u *ExtratoUsecaseImpl) ListarImportacoes(ctx context.Context, input portsin.ListarImportacoesInput) ([]*domaincaixa.ImportacaoExtrato, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	filtro := portsout.ImportacaoFiltro{PaginacaoFiltro: input.Filtro.Normalizada()}
	if input.ContaBancariaID != nil {
		conta, err := u.contaComAcesso(ctx, *input.ContaBancariaID, input.UsuarioID)
		if err != nil {
			return nil, err
		}
		filtro.ContaBancariaID = &conta.ID
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}
	filtro.EmpresaIDs = empresaIDs

	return u.extratos.ListarImportacoes(ctx, filtro)
}

func (u *ExtratoUsecaseImpl) DesfazerImportacao(ctx context.Context, importacaoID, usuarioID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	importacao, err := u.extratos.ObterImportacao(ctx, importacaoID)
	if err != nil {
		return err
	}
	if importacao == nil {
		return domaincaixa.ErroImportacaoNaoEncontrada()
	}
	if _, err := u.contaComAcesso(ctx, importacao.ContaBancariaID, usuarioID); err != nil {
		return err
	}

	return u.extratos.DesfazerImportacao(ctx, importacao)
}

func (u *ExtratoUsecaseImpl) ListarLancamentos(ctx context.Context, input portsin.ListarLancamentosInput) ([]*domaincaixa.LancamentoExtrato, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	conta, err := u.contaComAcesso(ctx, input.ContaBancariaID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	de, ate, err := intervaloDeDatas(input.De, input.Ate)
	if err != nil {
		return nil, err
	}

	filtro := portsout.LancamentoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		ContaBancariaID: conta.ID,
		De:              de,
		Ate:             ate,
	}
	if input.Situacao != "" {
		situacao, err := domaincaixa.NovaSituacaoLancamento(input.Situacao)
		if err != nil {
			return nil, err
		}
		filtro.Situacao = &situacao
	}

	return u.extratos.ListarLancamentos(ctx, filtro)
}

func (u *ExtratoUsecaseImpl) contaDoExtrato(
	ctx context.Context,
	extrato *domaincaixa.ExtratoOFX,
	contaID *uuid.UUID,
	usuarioID uuid.UUID,
) (*domainfinanceiro.ContaBancaria, error) {
	if contaID != nil {
		conta, err := u.contaComAcesso(ctx, *contaID, usuarioID)
		if err != nil {
			return nil, err
		}
		if !extrato.IdentificaConta() {
			return conta, nil
		}

		grau := extrato.Correspondencia(conta)
		if grau == domaincaixa.SemCorrespondencia {
			return nil, erroDeOutraConta(extrato)
		}
		if grau < domaincaixa.CorrespondenciaExata && extrato.Banco() != "" {
			outras, err := u.extratos.ContasDoBanco(ctx, domainfinanceiro.BancoDe(extrato.Banco()))
			if err != nil {
				return nil, err
			}
			for _, outra := range outras {
				if outra.ID != conta.ID && extrato.Correspondencia(outra) > grau {
					return nil, erroDeOutraConta(extrato)
				}
			}
		}

		return conta, nil
	}

	banco := extrato.Banco()
	if banco == "" || !extrato.IdentificaConta() {
		return nil, domain.ErroValidacao("o extrato não identifica a conta; informe conta_bancaria_id")
	}

	candidatas, err := u.extratos.ContasDoBanco(ctx, domainfinanceiro.BancoDe(banco))
	if err != nil {
		return nil, err
	}
	acesso, err := u.acesso(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	melhorGrau := domaincaixa.SemCorrespondencia
	var encontradas []*domainfinanceiro.ContaBancaria
	for _, conta := range candidatas {
		if !conta.Ativa || !acesso.Permite(conta.EmpresaID) {
			continue
		}
		grau := extrato.Correspondencia(conta)
		switch {
		case grau == domaincaixa.SemCorrespondencia || grau < melhorGrau:
		case grau > melhorGrau:
			melhorGrau = grau
			encontradas = []*domainfinanceiro.ContaBancaria{conta}
		default:
			encontradas = append(encontradas, conta)
		}
	}

	switch len(encontradas) {
	case 0:
		return nil, domain.ErroValidacao("nenhuma conta ativa no seu acesso corresponde ao extrato (" + identificacaoDoExtrato(extrato) +
			"); cadastre a conta ou informe conta_bancaria_id")
	case 1:
		return encontradas[0], nil
	default:
		return nil, domain.ErroValidacao("mais de uma conta corresponde ao extrato; informe conta_bancaria_id")
	}
}

func erroDeOutraConta(extrato *domaincaixa.ExtratoOFX) error {
	return domain.ErroValidacao("o extrato é de outra conta (" + identificacaoDoExtrato(extrato) + ")")
}

func identificacaoDoExtrato(extrato *domaincaixa.ExtratoOFX) string {
	partes := []string{"banco " + vazioComoNaoInformado(extrato.BancoID)}
	partes = append(partes, "agência "+vazioComoNaoInformado(extrato.AgenciaID))
	partes = append(partes, "conta "+vazioComoNaoInformado(extrato.ContaID))

	return strings.Join(partes, ", ")
}

func vazioComoNaoInformado(valor string) string {
	if strings.TrimSpace(valor) == "" {
		return "não informada"
	}

	return strings.TrimSpace(valor)
}

func intervaloDeDatas(de, ate *time.Time) (*time.Time, *time.Time, error) {
	if de != nil {
		data := domaincaixa.DataDe(*de)
		de = &data
	}
	if ate != nil {
		data := domaincaixa.DataDe(*ate)
		ate = &data
	}
	if de != nil && ate != nil && ate.Before(*de) {
		return nil, nil, domain.ErroValidacao("data final anterior à inicial")
	}

	return de, ate, nil
}
