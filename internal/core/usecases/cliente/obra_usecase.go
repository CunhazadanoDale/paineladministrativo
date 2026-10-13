package cliente

import (
	"context"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
)

var _ portsin.ObraUseCase = (*ObraUsecaseImpl)(nil)

type ObraUsecaseImpl struct {
	obras    portsout.ObraRepository
	clientes portsout.ClienteRepository
	permissoes
}

func NewObraUsecase(
	obras portsout.ObraRepository,
	clientes portsout.ClienteRepository,
	representantes portsout.RepresentanteRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ObraUsecaseImpl {
	return &ObraUsecaseImpl{
		obras:    obras,
		clientes: clientes,
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
		},
	}
}

func (u *ObraUsecaseImpl) Criar(ctx context.Context, input portsin.CriarObraInput) (*domaincliente.Obra, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	cliente, err := u.clienteVisivel(ctx, perfil, input.ClienteID)
	if err != nil {
		return nil, err
	}
	if !cliente.Ativo {
		return nil, domain.ErroValidacao("cliente inativo não recebe obra nova")
	}

	dados, err := dadosDaObra(input.Dados)
	if err != nil {
		return nil, err
	}

	obra, err := domaincliente.NovaObra(cliente.ID, dados)
	if err != nil {
		return nil, err
	}

	if _, err := u.obras.Criar(ctx, obra); err != nil {
		return nil, traduzirObraRepetida(err)
	}

	return obra, nil
}

func (u *ObraUsecaseImpl) Obter(ctx context.Context, obraID, usuarioID uuid.UUID) (*domaincliente.Obra, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	return u.buscarVisivel(ctx, perfil, obraID)
}

func (u *ObraUsecaseImpl) Listar(ctx context.Context, input portsin.ListarObrasInput) ([]*domaincliente.Obra, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	cliente, err := u.clienteVisivel(ctx, perfil, input.ClienteID)
	if err != nil {
		return nil, err
	}

	return u.obras.Listar(ctx, portsout.ObraFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		ClienteID:       cliente.ID,
		Ativo:           input.Ativo,
	})
}

func (u *ObraUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarObraInput) (*domaincliente.Obra, error) {
	return u.alterar(ctx, input.ObraID, input.UsuarioID, func(obra *domaincliente.Obra, agora time.Time) error {
		dados, err := dadosDaObra(input.Dados)
		if err != nil {
			return err
		}

		return obra.Alterar(dados, agora)
	})
}

func (u *ObraUsecaseImpl) Ativar(ctx context.Context, obraID, usuarioID uuid.UUID) (*domaincliente.Obra, error) {
	return u.alterar(ctx, obraID, usuarioID, func(obra *domaincliente.Obra, agora time.Time) error {
		obra.Ativar(agora)
		return nil
	})
}

func (u *ObraUsecaseImpl) Desativar(ctx context.Context, obraID, usuarioID uuid.UUID) (*domaincliente.Obra, error) {
	return u.alterar(ctx, obraID, usuarioID, func(obra *domaincliente.Obra, agora time.Time) error {
		obra.Desativar(agora)
		return nil
	})
}

func (u *ObraUsecaseImpl) alterar(
	ctx context.Context,
	obraID, usuarioID uuid.UUID,
	aplicar func(*domaincliente.Obra, time.Time) error,
) (*domaincliente.Obra, error) {
	perfil, err := u.exigeComercial(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	obra, err := u.buscarVisivel(ctx, perfil, obraID)
	if err != nil {
		return nil, err
	}

	if err := aplicar(obra, time.Now().UTC()); err != nil {
		return nil, err
	}

	atualizado, err := u.obras.Atualizar(ctx, obra)
	if err != nil {
		return nil, traduzirObraRepetida(err)
	}
	if !atualizado {
		return nil, erroObraNaoEncontrada()
	}

	return obra, nil
}

func (u *ObraUsecaseImpl) buscarVisivel(ctx context.Context, perfil perfilDoUsuario, obraID uuid.UUID) (*domaincliente.Obra, error) {
	obra, err := u.obras.Obter(ctx, obraID)
	if err != nil {
		return nil, err
	}
	if obra == nil {
		return nil, erroObraNaoEncontrada()
	}
	if _, err := u.clienteVisivel(ctx, perfil, obra.ClienteID); err != nil {
		return nil, err
	}

	return obra, nil
}

func (u *ObraUsecaseImpl) clienteVisivel(ctx context.Context, perfil perfilDoUsuario, clienteID uuid.UUID) (*domaincliente.Cliente, error) {
	cliente, err := u.clientes.Obter(ctx, clienteID)
	if err != nil {
		return nil, err
	}
	if cliente == nil {
		return nil, erroClienteNaoEncontrado()
	}
	if !perfil.podeVer(cliente) {
		return nil, erroClienteDeOutroRepresentante()
	}

	return cliente, nil
}

func dadosDaObra(input portsin.DadosObraInput) (domaincliente.DadosObra, error) {
	uf, err := domainempresa.NovaUF(input.UF)
	if err != nil {
		return domaincliente.DadosObra{}, err
	}

	dados := domaincliente.DadosObra{
		Nome: input.Nome,
		Endereco: domaincliente.Endereco{
			Logradouro:  input.Logradouro,
			Numero:      input.Numero,
			Complemento: input.Complemento,
			Bairro:      input.Bairro,
			Cidade:      input.Cidade,
			UF:          uf,
			CEP:         input.CEP,
		},
	}

	if input.CNPJ != nil && *input.CNPJ != "" {
		cnpj, err := domainempresa.NovoCNPJ(*input.CNPJ)
		if err != nil {
			return dados, domain.ErroValidacao("cnpj da obra inválido")
		}
		dados.CNPJ = &cnpj
	}

	return dados, nil
}

func erroObraNaoEncontrada() error {
	return domain.ErroNaoEncontrado("obra não encontrada")
}

func traduzirObraRepetida(err error) error {
	if errors.Is(err, domain.ErrConflito) {
		return domain.ErroConflito("já existe uma obra com este nome no cliente")
	}

	return err
}
