package cliente

import (
	"context"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
)

var _ portsin.RepresentanteUseCase = (*RepresentanteUsecaseImpl)(nil)

type RepresentanteUsecaseImpl struct {
	permissoes
}

func NewRepresentanteUsecase(
	representantes portsout.RepresentanteRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *RepresentanteUsecaseImpl {
	return &RepresentanteUsecaseImpl{
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
		},
	}
}

func (u *RepresentanteUsecaseImpl) Criar(ctx context.Context, input portsin.CriarRepresentanteInput) (*domaincliente.Representante, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, uuid.Nil, input.Dados)
	if err != nil {
		return nil, err
	}

	representante, err := domaincliente.NovoRepresentante(dados)
	if err != nil {
		return nil, err
	}

	if _, err := u.representantes.Criar(ctx, representante); err != nil {
		return nil, traduzirRepresentanteRepetido(err)
	}

	return representante, nil
}

func (u *RepresentanteUsecaseImpl) Obter(ctx context.Context, representanteID, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if perfil.restritoAoRepresentante() && *perfil.representanteID != representanteID {
		return nil, domain.ErroPermissao("representante só consulta o próprio cadastro")
	}

	return u.buscar(ctx, representanteID)
}

func (u *RepresentanteUsecaseImpl) Listar(ctx context.Context, input portsin.ListarRepresentantesInput) ([]*domaincliente.Representante, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if perfil.restritoAoRepresentante() {
		if input.Filtro.Normalizada().Page > 1 {
			return []*domaincliente.Representante{}, nil
		}
		proprio, err := u.buscar(ctx, *perfil.representanteID)
		if err != nil {
			return nil, err
		}

		return []*domaincliente.Representante{proprio}, nil
	}

	return u.representantes.Listar(ctx, portsout.RepresentanteFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Busca:           input.Busca,
		Ativo:           input.Ativo,
	})
}

func (u *RepresentanteUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarRepresentanteInput) (*domaincliente.Representante, error) {
	return u.alterar(ctx, input.RepresentanteID, input.UsuarioID, func(representante *domaincliente.Representante, agora time.Time) error {
		dados, err := u.dadosValidados(ctx, representante.ID, input.Dados)
		if err != nil {
			return err
		}

		return representante.Alterar(dados, agora)
	})
}

func (u *RepresentanteUsecaseImpl) Ativar(ctx context.Context, representanteID, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	return u.alterar(ctx, representanteID, usuarioID, func(representante *domaincliente.Representante, agora time.Time) error {
		representante.Ativar(agora)
		return nil
	})
}

func (u *RepresentanteUsecaseImpl) Desativar(ctx context.Context, representanteID, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	return u.alterar(ctx, representanteID, usuarioID, func(representante *domaincliente.Representante, agora time.Time) error {
		representante.Desativar(agora)
		return nil
	})
}

func (u *RepresentanteUsecaseImpl) DefinirFornecedor(ctx context.Context, input portsin.DefinirFornecedorInput) (*domaincliente.Representante, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	representante, err := u.buscar(ctx, input.RepresentanteID)
	if err != nil {
		return nil, err
	}

	representante.DefinirFornecedor(input.FornecedorID, time.Now().UTC())
	atualizado, err := u.representantes.DefinirFornecedor(ctx, representante)
	if err != nil {
		return nil, err
	}
	if !atualizado {
		return nil, erroRepresentanteNaoEncontrado()
	}

	return representante, nil
}

func (u *RepresentanteUsecaseImpl) alterar(
	ctx context.Context,
	representanteID, usuarioID uuid.UUID,
	aplicar func(*domaincliente.Representante, time.Time) error,
) (*domaincliente.Representante, error) {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return nil, err
	}

	representante, err := u.buscar(ctx, representanteID)
	if err != nil {
		return nil, err
	}

	if err := aplicar(representante, time.Now().UTC()); err != nil {
		return nil, err
	}

	atualizado, err := u.representantes.Atualizar(ctx, representante)
	if err != nil {
		return nil, traduzirRepresentanteRepetido(err)
	}
	if !atualizado {
		return nil, erroRepresentanteNaoEncontrado()
	}

	return representante, nil
}

func (u *RepresentanteUsecaseImpl) buscar(ctx context.Context, representanteID uuid.UUID) (*domaincliente.Representante, error) {
	representante, err := u.representantes.Obter(ctx, representanteID)
	if err != nil {
		return nil, err
	}
	if representante == nil {
		return nil, erroRepresentanteNaoEncontrado()
	}

	return representante, nil
}

func (u *RepresentanteUsecaseImpl) dadosValidados(ctx context.Context, proprioID uuid.UUID, input portsin.DadosRepresentanteInput) (domaincliente.DadosRepresentante, error) {
	dados := domaincliente.DadosRepresentante{
		Nome:     input.Nome,
		Email:    input.Email,
		Telefone: input.Telefone,
	}

	var err error
	if dados.ComissaoProntaEntrega, err = domaincliente.NovoPercentual(input.ComissaoProntaEntregaBP); err != nil {
		return dados, err
	}
	if dados.ComissaoImportacao, err = domaincliente.NovoPercentual(input.ComissaoImportacaoBP); err != nil {
		return dados, err
	}

	if input.Documento != nil && *input.Documento != "" {
		documento, err := domaincliente.NovoDocumento(*input.Documento)
		if err != nil {
			return dados, err
		}
		existente, err := u.representantes.ObterPorDocumento(ctx, documento.Valor())
		if err != nil {
			return dados, err
		}
		if existente != nil && existente.ID != proprioID {
			return dados, domain.ErroConflito("já existe um representante com este documento")
		}
		dados.Documento = &documento
	}

	if input.UsuarioVinculadoID != nil && *input.UsuarioVinculadoID != uuid.Nil {
		usuario, err := u.usuarios.Obter(ctx, *input.UsuarioVinculadoID)
		if err != nil {
			return dados, err
		}
		if usuario == nil {
			return dados, domain.ErroValidacao("usuário vinculado não encontrado")
		}
		existente, err := u.representantes.ObterPorUsuario(ctx, usuario.ID)
		if err != nil {
			return dados, err
		}
		if existente != nil && existente.ID != proprioID {
			return dados, domain.ErroConflito("usuário já vinculado a outro representante")
		}
		dados.UsuarioID = &usuario.ID
	}

	return dados, nil
}

func erroRepresentanteNaoEncontrado() error {
	return domain.ErroNaoEncontrado("representante não encontrado")
}

func traduzirRepresentanteRepetido(err error) error {
	if errors.Is(err, domain.ErrConflito) {
		return domain.ErroConflito("já existe um representante com este documento ou usuário")
	}

	return err
}
