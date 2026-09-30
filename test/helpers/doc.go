// Package helpers concentra a infraestrutura compartilhada pelos testes que
// saem do pacote unitário: banco de teste descartável por teste, aplicação
// das migrations, fixtures e montagem do servidor HTTP.
//
// Convenção do projeto:
//
//   - teste unitário fica ao lado do código que ele testa (foo_test.go);
//   - tudo que precisa de infraestrutura (banco, rede, servidor) vem para
//     test/, espelhando a árvore intern/.
package helpers
