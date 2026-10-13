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

var _ portsin.ClienteUseCase = (*ClienteUsecaseImpl)(nil)

type ClienteUsecaseImpl struct {
	clientes portsout.ClienteRepository
	grupos   portsout.GrupoRepository
	empresas portsout.EmpresaRepository
	permissoes
}

func NewClienteUsecase(
	clientes portsout.ClienteRepository,
	grupos portsout.GrupoRepository,
	representantes portsout.RepresentanteRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ClienteUsecaseImpl {
	return &ClienteUsecaseImpl{
		clientes: clientes,
		grupos:   grupos,
		empresas: empresas,
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
		},
	}
}

func (u *ClienteUsecaseImpl) Criar(ctx context.Context, input portsin.CriarClienteInput) (*domaincliente.Cliente, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	documento, err := domaincliente.NovoDocumento(input.Documento)
	if err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, perfil, nil, input.Dados)
	if err != nil {
		return nil, err
	}

	existente, err := u.clientes.ObterPorDocumento(ctx, documento.Valor())
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, erroDocumentoRepetido()
	}

	cliente, err := domaincliente.NovoCliente(documento, dados, nil)
	if err != nil {
		return nil, err
	}

	if _, err := u.clientes.Criar(ctx, cliente); err != nil {
		return nil, traduzirClienteRepetido(err)
	}

	return cliente, nil
}

func (u *ClienteUsecaseImpl) Obter(ctx context.Context, clienteID, usuarioID uuid.UUID) (*domaincliente.Cliente, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	return u.buscarVisivel(ctx, perfil, clienteID)
}

func (u *ClienteUsecaseImpl) Listar(ctx context.Context, input portsin.ListarClientesInput) ([]*domaincliente.Cliente, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	filtro := portsout.ClienteFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Busca:           input.Busca,
		GrupoID:         input.GrupoID,
		RepresentanteID: input.RepresentanteID,
		Ativo:           input.Ativo,
		LeadID:          input.LeadID,
	}

	if perfil.restritoAoRepresentante() {
		if input.RepresentanteID != nil && *input.RepresentanteID != *perfil.representanteID {
			return nil, erroClienteDeOutroRepresentante()
		}
		filtro.RepresentanteID = perfil.representanteID
	}
	if input.Tipo != "" {
		if filtro.Tipo, err = domaincliente.NovoTipoCliente(input.Tipo); err != nil {
			return nil, err
		}
	}
	if input.UF != "" {
		uf, err := domainempresa.NovaUF(input.UF)
		if err != nil {
			return nil, err
		}
		filtro.UF = uf.String()
	}

	return u.clientes.Listar(ctx, filtro)
}

func (u *ClienteUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarClienteInput) (*domaincliente.Cliente, error) {
	return u.alterar(ctx, input.ClienteID, input.UsuarioID, func(perfil perfilDoUsuario, cliente *domaincliente.Cliente, agora time.Time) error {
		dados, err := u.dadosValidados(ctx, perfil, cliente, input.Dados)
		if err != nil {
			return err
		}

		return cliente.Alterar(dados, agora)
	})
}

func (u *ClienteUsecaseImpl) Ativar(ctx context.Context, clienteID, usuarioID uuid.UUID) (*domaincliente.Cliente, error) {
	return u.alterar(ctx, clienteID, usuarioID, func(_ perfilDoUsuario, cliente *domaincliente.Cliente, agora time.Time) error {
		cliente.Ativar(agora)
		return nil
	})
}

func (u *ClienteUsecaseImpl) Desativar(ctx context.Context, clienteID, usuarioID uuid.UUID) (*domaincliente.Cliente, error) {
	return u.alterar(ctx, clienteID, usuarioID, func(_ perfilDoUsuario, cliente *domaincliente.Cliente, agora time.Time) error {
		cliente.Desativar(agora)
		return nil
	})
}

func (u *ClienteUsecaseImpl) Remover(ctx context.Context, clienteID, usuarioID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	removido, err := u.clientes.Remover(ctx, clienteID)
	if err != nil {
		return err
	}
	if !removido {
		return erroClienteNaoEncontrado()
	}

	return nil
}

func (u *ClienteUsecaseImpl) alterar(
	ctx context.Context,
	clienteID, usuarioID uuid.UUID,
	aplicar func(perfilDoUsuario, *domaincliente.Cliente, time.Time) error,
) (*domaincliente.Cliente, error) {
	perfil, err := u.exigeComercial(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	cliente, err := u.buscarVisivel(ctx, perfil, clienteID)
	if err != nil {
		return nil, err
	}

	if err := aplicar(perfil, cliente, time.Now().UTC()); err != nil {
		return nil, err
	}

	atualizado, err := u.clientes.Atualizar(ctx, cliente)
	if err != nil {
		return nil, traduzirClienteRepetido(err)
	}
	if !atualizado {
		return nil, erroClienteNaoEncontrado()
	}

	return cliente, nil
}

func (u *ClienteUsecaseImpl) buscarVisivel(ctx context.Context, perfil perfilDoUsuario, clienteID uuid.UUID) (*domaincliente.Cliente, error) {
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

func (u *ClienteUsecaseImpl) dadosValidados(
	ctx context.Context,
	perfil perfilDoUsuario,
	atual *domaincliente.Cliente,
	input portsin.DadosClienteInput,
) (domaincliente.DadosCliente, error) {
	dados := domaincliente.DadosCliente{
		Nome:                    input.Nome,
		NomeFantasia:            input.NomeFantasia,
		Cidade:                  input.Cidade,
		EmailNF:                 input.EmailNF,
		Telefone:                input.Telefone,
		Observacoes:             input.Observacoes,
		ExigeNumeroPedidoNaNF:   input.ExigeNumeroPedidoNaNF,
		ProibeAntecipacao:       input.ProibeAntecipacao,
		GrupoID:                 idInformado(input.GrupoID),
		EmpresaRelacionamentoID: idInformado(input.EmpresaRelacionamentoID),
		RepresentanteID:         idInformado(input.RepresentanteID),
	}

	var err error
	if dados.Tipo, err = domaincliente.NovoTipoCliente(input.Tipo); err != nil {
		return dados, err
	}
	if dados.UF, err = domainempresa.NovaUF(input.UF); err != nil {
		return dados, err
	}

	if perfil.restritoAoRepresentante() {
		if dados.RepresentanteID != nil && *dados.RepresentanteID != *perfil.representanteID {
			return dados, domain.ErroPermissao("representante não troca o representante do cliente")
		}
		dados.RepresentanteID = perfil.representanteID
	}

	var atualGrupo, atualRepresentante, atualEmpresa *uuid.UUID
	if atual != nil {
		atualGrupo, atualRepresentante, atualEmpresa = atual.GrupoID, atual.RepresentanteID, atual.EmpresaRelacionamentoID
	}

	if err := u.validarGrupo(ctx, dados.GrupoID, atualGrupo); err != nil {
		return dados, err
	}
	if err := u.validarRepresentante(ctx, dados.RepresentanteID, atualRepresentante); err != nil {
		return dados, err
	}
	if err := u.validarEmpresa(ctx, dados.EmpresaRelacionamentoID, atualEmpresa); err != nil {
		return dados, err
	}

	return dados, nil
}

func (u *ClienteUsecaseImpl) validarGrupo(ctx context.Context, grupoID, atual *uuid.UUID) error {
	if grupoID == nil {
		return nil
	}

	grupo, err := u.grupos.Obter(ctx, *grupoID)
	if err != nil {
		return err
	}
	if grupo == nil {
		return domain.ErroValidacao("grupo de clientes não encontrado")
	}
	if !grupo.Ativo && !mesmoID(grupoID, atual) {
		return domain.ErroValidacao("grupo de clientes inativo não recebe cliente")
	}

	return nil
}

func (u *ClienteUsecaseImpl) validarRepresentante(ctx context.Context, representanteID, atual *uuid.UUID) error {
	if representanteID == nil {
		return nil
	}

	representante, err := u.representantes.Obter(ctx, *representanteID)
	if err != nil {
		return err
	}
	if representante == nil {
		return domain.ErroValidacao("representante não encontrado")
	}
	if !representante.Ativo && !mesmoID(representanteID, atual) {
		return domain.ErroValidacao("representante inativo não recebe cliente")
	}

	return nil
}

func (u *ClienteUsecaseImpl) validarEmpresa(ctx context.Context, empresaID, atual *uuid.UUID) error {
	if empresaID == nil {
		return nil
	}

	empresa, err := u.empresas.Obter(ctx, *empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domain.ErroValidacao("empresa de relacionamento não encontrada")
	}
	if !empresa.Ativa && !mesmoID(empresaID, atual) {
		return domain.ErroValidacao("empresa de relacionamento inativa")
	}

	return nil
}

func idInformado(id *uuid.UUID) *uuid.UUID {
	if id == nil || *id == uuid.Nil {
		return nil
	}

	return id
}

func mesmoID(a, b *uuid.UUID) bool {
	return a != nil && b != nil && *a == *b
}

func erroClienteNaoEncontrado() error {
	return domain.ErroNaoEncontrado("cliente não encontrado")
}

func erroDocumentoRepetido() error {
	return domain.ErroConflito("já existe um cliente com este documento")
}

func traduzirClienteRepetido(err error) error {
	if errors.Is(err, domain.ErrConflito) {
		return erroDocumentoRepetido()
	}

	return err
}
