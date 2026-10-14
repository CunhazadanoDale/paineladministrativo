package financeiro

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/financeiro"
	"github.com/google/uuid"
)

var _ portsin.ContaBancariaUseCase = (*ContaBancariaUsecaseImpl)(nil)

type ContaBancariaUsecaseImpl struct {
	contas portsout.ContaBancariaRepository
	permissoes
	escopoPorEmpresa
}

func NewContaBancariaUsecase(
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ContaBancariaUsecaseImpl {
	return &ContaBancariaUsecaseImpl{
		contas:           contas,
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas},
	}
}

func (u *ContaBancariaUsecaseImpl) Criar(ctx context.Context, input portsin.CriarContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}
	if err := u.validarEmpresa(ctx, input.UsuarioID, input.EmpresaID); err != nil {
		return nil, err
	}

	dados, err := dadosDaConta(input)
	if err != nil {
		return nil, err
	}

	conta, err := domainfinanceiro.NovaContaBancaria(dados)
	if err != nil {
		return nil, err
	}

	if err := u.contas.Criar(ctx, conta); err != nil {
		return nil, err
	}

	return conta, nil
}

func (u *ContaBancariaUsecaseImpl) Obter(ctx context.Context, contaID, usuarioID uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	return u.buscarComAcesso(ctx, contaID, usuarioID)
}

func (u *ContaBancariaUsecaseImpl) Listar(ctx context.Context, input portsin.ListarContasBancariasInput) ([]*domainfinanceiro.ContaBancaria, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	filtro := portsout.ContaBancariaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		EmpresaIDs:      empresaIDs,
		Ativa:           input.Ativa,
	}
	if input.Banco != "" {
		banco, err := domainfinanceiro.NovoBanco(input.Banco)
		if err != nil {
			return nil, err
		}
		filtro.Banco = &banco
	}
	if input.Tipo != "" {
		tipo, err := domainfinanceiro.NovoTipoConta(input.Tipo)
		if err != nil {
			return nil, err
		}
		filtro.Tipo = &tipo
	}

	return u.contas.Listar(ctx, filtro)
}

func (u *ContaBancariaUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	conta, err := u.buscarParaAlterar(ctx, input.ContaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	if err := conta.AlterarApelido(input.Apelido, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := u.contas.Atualizar(ctx, conta); err != nil {
		return nil, err
	}

	return conta, nil
}

func (u *ContaBancariaUsecaseImpl) Ativar(ctx context.Context, input portsin.MudarSituacaoContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	conta, err := u.buscarParaAlterar(ctx, input.ContaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	if !conta.Ativar(time.Now().UTC()) {
		return conta, nil
	}
	if err := u.contas.Ativar(ctx, conta); err != nil {
		return nil, err
	}

	return conta, nil
}

func (u *ContaBancariaUsecaseImpl) Desativar(ctx context.Context, input portsin.MudarSituacaoContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	conta, err := u.buscarParaAlterar(ctx, input.ContaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	if !conta.Desativar(time.Now().UTC()) {
		return conta, nil
	}
	if err := u.contas.Desativar(ctx, conta); err != nil {
		return nil, err
	}

	return conta, nil
}

func (u *ContaBancariaUsecaseImpl) Remover(ctx context.Context, contaID, usuarioID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	conta, err := u.buscarComAcesso(ctx, contaID, usuarioID)
	if err != nil {
		return err
	}

	return u.contas.Remover(ctx, conta.ID)
}

func (u *ContaBancariaUsecaseImpl) Bancos() []domainfinanceiro.Banco {
	return domainfinanceiro.Bancos()
}

func (u *ContaBancariaUsecaseImpl) validarEmpresa(ctx context.Context, usuarioID, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa da conta bancária não informada")
	}
	if err := u.exigeAcesso(ctx, usuarioID, empresaID); err != nil {
		return err
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domainfinanceiro.ErroEmpresaDaContaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domainfinanceiro.ErroEmpresaInativa()
	}

	return nil
}

func (u *ContaBancariaUsecaseImpl) buscarParaAlterar(ctx context.Context, contaID, usuarioID uuid.UUID, versao int) (*domainfinanceiro.ContaBancaria, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	conta, err := u.buscarComAcesso(ctx, contaID, usuarioID)
	if err != nil {
		return nil, err
	}
	if err := conta.ConferirVersao(versao); err != nil {
		return nil, err
	}

	return conta, nil
}

func (u *ContaBancariaUsecaseImpl) buscarComAcesso(ctx context.Context, contaID, usuarioID uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	conta, err := u.contas.Obter(ctx, contaID)
	if err != nil {
		return nil, err
	}
	if conta == nil {
		return nil, domainfinanceiro.ErroContaBancariaNaoEncontrada()
	}
	if err := u.exigeAcesso(ctx, usuarioID, conta.EmpresaID); err != nil {
		return nil, err
	}

	return conta, nil
}

func dadosDaConta(input portsin.CriarContaBancariaInput) (domainfinanceiro.DadosContaBancaria, error) {
	banco, err := domainfinanceiro.NovoBanco(input.Banco)
	if err != nil {
		return domainfinanceiro.DadosContaBancaria{}, err
	}

	tipo, err := domainfinanceiro.NovoTipoConta(input.Tipo)
	if err != nil {
		return domainfinanceiro.DadosContaBancaria{}, err
	}

	agencia, err := domainfinanceiro.NovaAgencia(input.Agencia, input.AgenciaDigito)
	if err != nil {
		return domainfinanceiro.DadosContaBancaria{}, err
	}

	conta, err := domainfinanceiro.NovaConta(input.Conta, input.ContaDigito)
	if err != nil {
		return domainfinanceiro.DadosContaBancaria{}, err
	}

	return domainfinanceiro.DadosContaBancaria{
		EmpresaID: input.EmpresaID,
		Banco:     banco,
		Tipo:      tipo,
		Agencia:   agencia,
		Conta:     conta,
		Apelido:   input.Apelido,
	}, nil
}
