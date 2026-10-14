package fornecedor

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/fornecedor"
	"github.com/google/uuid"
)

var _ portsin.FornecedorUseCase = (*FornecedorUsecaseImpl)(nil)

type FornecedorUsecaseImpl struct {
	permissoes
	fornecedores portsout.FornecedorRepository
}

func NewFornecedorUsecase(
	fornecedores portsout.FornecedorRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *FornecedorUsecaseImpl {
	return &FornecedorUsecaseImpl{
		permissoes:   permissoes{usuarios: usuarios, cargos: cargos},
		fornecedores: fornecedores,
	}
}

func (u *FornecedorUsecaseImpl) Criar(ctx context.Context, input portsin.CriarFornecedorInput) (*domainfornecedor.Fornecedor, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	pais, err := domainfornecedor.NovoPais(input.Pais)
	if err != nil {
		return nil, err
	}

	var documento domaincliente.Documento
	if input.Documento != nil && strings.TrimSpace(*input.Documento) != "" {
		if documento, err = domaincliente.NovoDocumento(*input.Documento); err != nil {
			return nil, err
		}
	}

	dados, err := dadosDoFornecedor(input.Dados)
	if err != nil {
		return nil, err
	}

	fornecedor, err := domainfornecedor.NovoFornecedor(pais, documento, dados)
	if err != nil {
		return nil, err
	}

	if err := u.conferirDocumentoLivre(ctx, fornecedor); err != nil {
		return nil, err
	}
	if err := u.conferirIdentificacaoLivre(ctx, fornecedor); err != nil {
		return nil, err
	}

	if _, err := u.fornecedores.Criar(ctx, fornecedor); err != nil {
		return nil, traduzirFornecedorRepetido(err)
	}

	return fornecedor, nil
}

func (u *FornecedorUsecaseImpl) Obter(ctx context.Context, fornecedorID uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	return u.buscar(ctx, fornecedorID)
}

func (u *FornecedorUsecaseImpl) Listar(ctx context.Context, input portsin.ListarFornecedoresInput) ([]*domainfornecedor.Fornecedor, error) {
	filtro := portsout.FornecedorFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Busca:           strings.TrimSpace(input.Busca),
		Ativo:           input.Ativo,
	}

	if strings.TrimSpace(input.Pais) != "" {
		pais, err := domainfornecedor.NovoPais(input.Pais)
		if err != nil {
			return nil, err
		}
		filtro.Pais = pais
	}
	if strings.TrimSpace(input.Moeda) != "" {
		moeda, err := domainfornecedor.NovaMoeda(input.Moeda)
		if err != nil {
			return nil, err
		}
		filtro.Moeda = moeda
	}

	return u.fornecedores.Listar(ctx, filtro)
}

func (u *FornecedorUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarFornecedorInput) (*domainfornecedor.Fornecedor, error) {
	return u.alterar(ctx, input.FornecedorID, input.UsuarioID, input.Versao, func(fornecedor *domainfornecedor.Fornecedor, agora time.Time) (bool, error) {
		dados, err := dadosDoFornecedor(input.Dados)
		if err != nil {
			return false, err
		}
		if err := fornecedor.Alterar(dados, agora); err != nil {
			return false, err
		}

		return true, u.conferirIdentificacaoLivre(ctx, fornecedor)
	})
}

func (u *FornecedorUsecaseImpl) Ativar(ctx context.Context, input portsin.SituacaoFornecedorInput) (*domainfornecedor.Fornecedor, error) {
	return u.alterar(ctx, input.FornecedorID, input.UsuarioID, input.Versao, func(fornecedor *domainfornecedor.Fornecedor, agora time.Time) (bool, error) {
		return fornecedor.Ativar(agora), nil
	})
}

func (u *FornecedorUsecaseImpl) Desativar(ctx context.Context, input portsin.SituacaoFornecedorInput) (*domainfornecedor.Fornecedor, error) {
	return u.alterar(ctx, input.FornecedorID, input.UsuarioID, input.Versao, func(fornecedor *domainfornecedor.Fornecedor, agora time.Time) (bool, error) {
		return fornecedor.Desativar(agora), nil
	})
}

func (u *FornecedorUsecaseImpl) Remover(ctx context.Context, fornecedorID, usuarioID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	removido, err := u.fornecedores.Remover(ctx, fornecedorID)
	if err != nil {
		return err
	}
	if !removido {
		return erroFornecedorNaoEncontrado()
	}

	return nil
}

func (u *FornecedorUsecaseImpl) alterar(
	ctx context.Context,
	fornecedorID, usuarioID uuid.UUID,
	versao int,
	aplicar func(*domainfornecedor.Fornecedor, time.Time) (bool, error),
) (*domainfornecedor.Fornecedor, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	fornecedor, err := u.buscar(ctx, fornecedorID)
	if err != nil {
		return nil, err
	}
	if err := fornecedor.ConferirVersao(versao); err != nil {
		return nil, err
	}

	mudou, err := aplicar(fornecedor, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if !mudou {
		return fornecedor, nil
	}

	atualizado, err := u.fornecedores.Atualizar(ctx, fornecedor)
	if err != nil {
		return nil, err
	}
	if !atualizado {
		return nil, erroFornecedorNaoEncontrado()
	}

	return fornecedor, nil
}

func (u *FornecedorUsecaseImpl) buscar(ctx context.Context, fornecedorID uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	fornecedor, err := u.fornecedores.Obter(ctx, fornecedorID)
	if err != nil {
		return nil, err
	}
	if fornecedor == nil {
		return nil, erroFornecedorNaoEncontrado()
	}

	return fornecedor, nil
}

func (u *FornecedorUsecaseImpl) conferirDocumentoLivre(ctx context.Context, fornecedor *domainfornecedor.Fornecedor) error {
	if fornecedor.Documento.Vazio() {
		return nil
	}

	existente, err := u.fornecedores.ObterPorDocumento(ctx, fornecedor.Documento.Valor())
	if err != nil {
		return err
	}
	if existente != nil && existente.ID != fornecedor.ID {
		return domain.ErroConflito("já existe um fornecedor com este documento")
	}

	return nil
}

func (u *FornecedorUsecaseImpl) conferirIdentificacaoLivre(ctx context.Context, fornecedor *domainfornecedor.Fornecedor) error {
	if fornecedor.IdentificacaoEstrangeira == nil {
		return nil
	}

	existente, err := u.fornecedores.ObterPorIdentificacaoEstrangeira(ctx, fornecedor.Pais, *fornecedor.IdentificacaoEstrangeira)
	if err != nil {
		return err
	}
	if existente != nil && existente.ID != fornecedor.ID {
		return domain.ErroConflito("já existe um fornecedor com esta identificação neste país")
	}

	return nil
}

func dadosDoFornecedor(input portsin.DadosFornecedorInput) (domainfornecedor.DadosFornecedor, error) {
	moeda, err := domainfornecedor.NovaMoeda(input.MoedaPadrao)
	if err != nil {
		return domainfornecedor.DadosFornecedor{}, err
	}

	dados := domainfornecedor.DadosFornecedor{
		IdentificacaoEstrangeira: input.IdentificacaoEstrangeira,
		Nome:                     input.Nome,
		NomeFantasia:             input.NomeFantasia,
		MoedaPadrao:              moeda,
		Cidade:                   input.Cidade,
		ContatoNome:              input.ContatoNome,
		Email:                    input.Email,
		Telefone:                 input.Telefone,
		Observacoes:              input.Observacoes,
	}
	if input.UF != nil {
		uf := domainempresa.UFDe(*input.UF)
		dados.UF = &uf
	}

	return dados, nil
}

func erroFornecedorNaoEncontrado() error {
	return domain.ErroNaoEncontrado("fornecedor não encontrado")
}

func traduzirFornecedorRepetido(err error) error {
	if errors.Is(err, domain.ErrConflito) {
		return domain.ErroConflito("já existe um fornecedor com este documento ou identificação")
	}

	return err
}
