package notafiscal

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
)

var _ portsin.EventoUseCase = (*EventoUsecaseImpl)(nil)

type EventoUsecaseImpl struct {
	geradorDeTitulos
	leitorDeXML
	permissoes
}

func NewEventoUsecase(
	notas portsout.NotaFiscalRepository,
	arquivos portsout.ArquivoRepository,
	storage portsout.Storage,
	empresas portsout.EmpresaRepository,
	clientes portsout.ClienteRepository,
	titulos portsout.TituloReceberRepository,
	representantes portsout.RepresentanteRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *EventoUsecaseImpl {
	return &EventoUsecaseImpl{
		geradorDeTitulos: geradorDeTitulos{
			titulos: titulos,
		},
		leitorDeXML: leitorDeXML{
			arquivos: arquivos,
			storage:  storage,
			vinculos: notas,
		},
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
			clientes:       clientes,
			notas:          notas,
		},
	}
}

func (u *EventoUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarEventoInput) (*portsin.ImportacaoEvento, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	conteudo, err := u.leitorDeXML.ler(ctx, input.ArquivoID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	documento, err := domainnotafiscal.LerEvento(conteudo)
	if err != nil {
		return nil, err
	}

	nota, err := u.notas.ObterPorChave(ctx, documento.Chave.Valor())
	if err != nil {
		return nil, err
	}
	if nota == nil {
		return nil, domain.ErroNaoEncontrado("nota fiscal do evento ainda não foi importada")
	}
	if !perfil.acesso.Permite(nota.EmpresaID) {
		return nil, erroSemAcessoAEmpresa()
	}

	agora := time.Now().UTC()
	versaoLida, statusAnterior := nota.Versao, nota.Status
	evento, err := nota.AplicarEvento(documento, input.ArquivoID, input.UsuarioID, agora)
	if err != nil {
		return nil, err
	}

	if err := u.notas.Atualizar(ctx, &portsout.Alteracao{
		Nota:           nota,
		VersaoLida:     versaoLida,
		StatusAnterior: statusAnterior,
		Evento:         evento,
	}); err != nil {
		return nil, err
	}

	importacao := &portsin.ImportacaoEvento{
		Evento: evento, Nota: nota, Avisos: []string{}, TitulosCancelados: []uuid.UUID{}, TitulosMarcados: []uuid.UUID{},
	}
	if evento.Tipo == domainnotafiscal.TipoEventoCancelamento && nota.PodeTerTitulos() {
		resultado, err := u.cancelar(ctx, nota, input.UsuarioID, agora)
		if err != nil {
			importacao.ErroDosTitulos = err
			importacao.Avisos = append(importacao.Avisos, portsin.AvisoTitulosPendentes)
		} else {
			importacao.TitulosCancelados, importacao.TitulosMarcados = resultado.Cancelados, resultado.Marcados
		}
	}

	return importacao, nil
}

func (u *EventoUsecaseImpl) Listar(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) ([]*domainnotafiscal.Evento, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	if _, err := u.notaVisivel(ctx, perfil, notaFiscalID); err != nil {
		return nil, err
	}

	return u.notas.ListarEventos(ctx, notaFiscalID)
}
