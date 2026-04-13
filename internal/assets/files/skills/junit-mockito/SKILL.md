---
name: junit-mockito
description: >
  JUnit 5 and Mockito testing standards, conventions, and patterns for Java Spring Boot projects.
  Use when writing, reviewing, or refactoring test code with JUnit and Mockito.
  Triggers: (1) Creating new test classes, (2) Writing or modifying unit tests,
  (3) Writing integration tests with @WebMvcTest or @DataJpaTest,
  (4) Reviewing test code for best practices, (5) Any task involving *Test.java files,
  (6) When asked to add tests for existing code.
---

# JUnit & Mockito Standards

## Core Principles

1. **BDD estructura**: Todo test sigue Given/When/Then con comentarios separadores
2. **BDDMockito siempre**: `given()`/`then().should()` — nunca `when()`/`verify()`
3. **AssertJ como assertion principal**: `assertThat()` — nunca `assertTrue`/`assertFalse`
4. **Object Mothers**: Factoria de datos de test — nunca datos hardcodeados repetidos
5. **Unitarios con `@ExtendWith(MockitoExtension.class)`** — nunca `@SpringBootTest` para unit tests

## Quick Reference

```
@ExtendWith(MockitoExtension.class)     -> Tests unitarios (Service, Validator, Mapper)
@WebMvcTest(Controller.class)           -> Tests de controller (HTTP, validacion Jakarta)
@DataJpaTest                            -> Tests de repository (solo si hay @Query custom)

@Mock                                   -> Dependencias del SUT
@InjectMocks                            -> System Under Test
@MockitoBean                            -> Mocks en tests de integracion Spring
@Captor                                 -> Captura de argumentos complejos

given(mock.method()).willReturn(value)   -> Configurar mock
then(mock).should().method()             -> Verificar interaccion
then(mock).should(never()).method()      -> Verificar que NO se llamo

assertThat(result).isEqualTo(expected)   -> Assertion basica
assertAll(...)                           -> Multiples assertions agrupadas
assertThatThrownBy(() -> ...)            -> Verificar excepciones
```

## Test Naming

```java
// Clase
{ClaseBajoTest}Test.java

// Metodo
should{Comportamiento}_When{Condicion}()

// DisplayName
@DisplayName("Should complete transfer when accounts are valid")
```

## Test Structure

```java
@ExtendWith(MockitoExtension.class)
@DisplayName("TransferService")
class TransferServiceTest {

    @Mock private TransferRepository transferRepository;
    @Mock private TransferValidator transferValidator;
    @InjectMocks private TransferService transferService;

    @Nested
    @DisplayName("execute")
    class Execute {
        @Test
        @DisplayName("Should complete transfer when valid")
        void shouldCompleteTransfer_WhenValid() {
            // Given — preparar datos con Object Mothers + configurar mocks
            // When  — una sola accion
            // Then  — assertThat + then().should()
        }
    }
}
```

## Prohibido

| Antipatron | Alternativa correcta |
|------------|---------------------|
| `@SpringBootTest` para unitarios | `@ExtendWith(MockitoExtension.class)` |
| `when()` / `verify()` de Mockito | `given()` / `then().should()` de BDDMockito |
| `assertTrue(x)` / `assertFalse(x)` | `assertThat(x).isTrue()` / `assertThat(x).isFalse()` |
| Datos hardcodeados repetidos | Object Mothers |
| `@Test` sin `@DisplayName` | Siempre `@DisplayName` descriptivo |
| `try/catch` en tests | `assertThatThrownBy(() -> ...)` |
| `@MockBean` (deprecado) | `@MockitoBean` |
| Mock de value objects | Solo mockear dependencias con comportamiento |

## Detailed References

- **Testing patterns** (BDD, Object Mothers, BDDMockito, Assertions, Parameterized, Nested): see [references/patterns.md](references/patterns.md)
- **Conventions** (Naming, File structure, Antipatterns, Coverage strategy, Spring integration tests): see [references/conventions.md](references/conventions.md)
