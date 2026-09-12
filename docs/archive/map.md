# Code Map

## cmd/archivist/main.go

- `main` (function_declaration) line 10

## internal/archive/service.go

- `Service` (type_spec) line 15
- `error` (method_declaration) line 53
- `error` (method_declaration) line 73
- `string` (method_declaration) line 118
- `string` (method_declaration) line 134
- `slugify` (function_declaration) line 142

## internal/check/check.go

- `Options` (type_spec) line 14
- `Match` (type_spec) line 22
- `Result` (type_spec) line 28
- `Run` (function_declaration) line 33
- `addMatch` (function_declaration) line 81
- `pathsFromDiff` (function_declaration) line 95
- `unique` (function_declaration) line 112

## internal/chunk/chunk.go

- `Type` (type_spec) line 13
- `Chunk` (type_spec) line 32
- `HashContent` (function_declaration) line 42
- `annotateContent` (function_declaration) line 49
- `NewChunk` (function_declaration) line 67
- `ClassifyFile` (function_declaration) line 79
- `Classify` (function_declaration) line 86
- `StampADRScope` (function_declaration) line 103
- `StampOrigin` (function_declaration) line 122
- `MatchAnyPattern` (function_declaration) line 140

## internal/chunk/chunk_test.go

- `TestSplitGenericMarkdown` (function_declaration) line 10
- `TestSplitGenericMarkdownSplitsWhenLarge` (function_declaration) line 27
- `TestExtractCommentChunks` (function_declaration) line 39
- `TestSplitGoFile` (function_declaration) line 56
- `TestSplitGoMethodIncludesParentAndDoc` (function_declaration) line 88
- `TestClassifyADR` (function_declaration) line 122
- `TestClassifyADRDoesNotMatchSiblingPrefix` (function_declaration) line 129
- `TestClassifyADRGlob` (function_declaration) line 139
- `TestClassifyGlobalWins` (function_declaration) line 145
- `TestClassifyRepoADR` (function_declaration) line 156
- `TestSplitFileStampsADRScope` (function_declaration) line 167
- `TestStampOrigin` (function_declaration) line 198
- `TestClassifyMDCAsDoc` (function_declaration) line 209
- `TestSplitADRStaysTogether` (function_declaration) line 215
- `TestMatchAnyPatternSkipGlobs` (function_declaration) line 232
- `TestExtractCommentChunksIgnoresAnnotation` (function_declaration) line 241

## internal/chunk/generic.go

- `SplitGeneric` (function_declaration) line 18
- `splitMarkdown` (function_declaration) line 35
- `splitBySize` (function_declaration) line 68
- `copyMeta` (function_declaration) line 109
- `tailBytes` (function_declaration) line 120
- `ExtractCommentChunks` (function_declaration) line 139
- `hasCommentTag` (function_declaration) line 165
- `isTagChar` (function_declaration) line 195
- `isLikelyComment` (function_declaration) line 199
- `IsBinary` (function_declaration) line 210

## internal/chunk/split.go

- `SplitFile` (function_declaration) line 9

## internal/chunk/treesitter.go

- `languageSpec` (type_spec) line 17
- `SplitWithTreeSitter` (function_declaration) line 37
- `filePreamble` (function_declaration) line 111
- `leadingCommentBlock` (function_declaration) line 145
- `isDocCommentLine` (function_declaration) line 163
- `semanticParent` (function_declaration) line 172
- `nodeName` (function_declaration) line 193
- `firstIdent` (function_declaration) line 203
- `findNodeType` (function_declaration) line 216
- `nodeText` (function_declaration) line 231

## internal/cmd/embed.go

- `newEmbedCmd` (function_declaration) line 10

## internal/cmd/export.go

- `newExportCmd` (function_declaration) line 10

## internal/cmd/index.go

- `newIndexCmd` (function_declaration) line 12

## internal/cmd/init.go

- `newInitCmd` (function_declaration) line 12
- `configExists` (function_declaration) line 40
- `applyInit` (function_declaration) line 45

## internal/cmd/init_test.go

- `TestApplyInitCreatesDecisionDirs` (function_declaration) line 11

## internal/cmd/mcp.go

- `newMCPCmd` (function_declaration) line 9

## internal/cmd/migrate.go

- `newMigrateCmd` (function_declaration) line 10

## internal/cmd/publish.go

- `newPublishCmd` (function_declaration) line 8

## internal/cmd/remember.go

- `newRememberCmd` (function_declaration) line 17
- `newUpdateCmd` (function_declaration) line 70
- `newRetireCmd` (function_declaration) line 118
- `newCheckCmd` (function_declaration) line 152

## internal/cmd/root.go

- `NewRoot` (function_declaration) line 21
- `newVersionCmd` (function_declaration) line 49
- `repoRoot` (function_declaration) line 60
- `loadEnv` (function_declaration) line 68
- `openStore` (function_declaration) line 80
- `openHomeStore` (function_declaration) line 84
- `openHomeStoreExisting` (function_declaration) line 92
- `resolveHomeStorePath` (function_declaration) line 107
- `openStores` (function_declaration) line 120
- `appendGitignore` (function_declaration) line 133
- `newSearchCmd` (function_declaration) line 155
- `newStatusCmd` (function_declaration) line 209
- `status` (type_spec) line 233

## internal/cmd/skills.go

- `newSkillsCmd` (function_declaration) line 10

## internal/cmd/version_test.go

- `TestVersionFlagAndCommand` (function_declaration) line 11

## internal/codemap/codemap.go

- `languageSpec` (type_spec) line 21
- `Result` (type_spec) line 48
- `Extract` (function_declaration) line 55
- `extractGeneric` (function_declaration) line 121
- `findPackage` (function_declaration) line 140
- `symbolName` (function_declaration) line 163
- `isExported` (function_declaration) line 177
- `firstDocLine` (function_declaration) line 185
- `cleanImport` (function_declaration) line 201
- `IsBinary` (function_declaration) line 210

## internal/config/config.go

- `Config` (type_spec) line 26
- `PublishConfig` (type_spec) line 33
- `PublishDestination` (type_spec) line 37
- `OllamaConfig` (type_spec) line 41
- `parseTimeout` (function_declaration) line 51
- `IndexConfig` (type_spec) line 62
- `ADRConfig` (type_spec) line 71
- `StoreConfig` (type_spec) line 76
- `Default` (function_declaration) line 81
- `Load` (function_declaration) line 109
- `error` (method_declaration) line 139
- `Save` (function_declaration) line 171
- `DataDir` (function_declaration) line 181
- `StorePath` (function_declaration) line 185
- `ArchivistHome` (function_declaration) line 193
- `GlobalStorePath` (function_declaration) line 204
- `pathUnderDir` (function_declaration) line 226
- `UserRecordsDir` (function_declaration) line 235
- `UserDecisionsDir` (function_declaration) line 244
- `VirtualUserADRPath` (function_declaration) line 249
- `IsUserGlobalPath` (function_declaration) line 258

## internal/config/config_test.go

- `fullConfig` (function_declaration) line 13
- `TestDefaultConfig` (function_declaration) line 44
- `TestLoadFullConfigFixture` (function_declaration) line 79
- `TestLoadFullConfigFromDisk` (function_declaration) line 96
- `TestSaveLoadRoundTripAllKeys` (function_declaration) line 117
- `TestLoadMissingConfigUsesDefaults` (function_declaration) line 181
- `TestLoadInvalidJSON` (function_declaration) line 192
- `TestLoadOmitsADRUsesDefault` (function_declaration) line 202
- `TestLoadHonorGitignoreFalse` (function_declaration) line 232
- `TestLoadLegacyADRPaths` (function_declaration) line 254
- `TestVirtualUserADRPath` (function_declaration) line 280
- `TestIsUserGlobalPath` (function_declaration) line 289
- `TestGlobalStorePath` (function_declaration) line 298
- `TestStorePath` (function_declaration) line 327
- `TestDataDir` (function_declaration) line 343
- `TestLoadEmptyStorePathFallsBackToDefault` (function_declaration) line 350

## internal/embed/embedder.go

- `Embedder` (type_spec) line 9
- `Client` (type_spec) line 14
- `HealthStatus` (type_spec) line 19
- `CheckHealth` (function_declaration) line 24

## internal/embed/ollama.go

- `OllamaClient` (type_spec) line 15
- `NewOllamaClient` (function_declaration) line 22
- `NewOllamaClientFromConfig` (function_declaration) line 26
- `NewOllamaClientWithTimeout` (function_declaration) line 30
- `error` (method_declaration) line 38
- `embedRequest` (type_spec) line 55
- `embedResponse` (type_spec) line 60
- `int` (method_declaration) line 94
- `FakeEmbedder` (type_spec) line 99
- `int` (method_declaration) line 122
- `error` (method_declaration) line 129

## internal/embed/worker.go

- `Worker` (type_spec) line 14
- `WorkerOptions` (type_spec) line 20
- `DefaultWorkerOptions` (function_declaration) line 26
- `error` (method_declaration) line 99
- `OpenWorkerStores` (function_declaration) line 127
- `EnqueueRecord` (function_declaration) line 147

## internal/export/export.go

- `Options` (type_spec) line 18
- `Manifest` (type_spec) line 23
- `Entry` (type_spec) line 29
- `Run` (function_declaration) line 43
- `collectRecords` (function_declaration) line 67
- `writeRecords` (function_declaration) line 87
- `writeIndex` (function_declaration) line 100
- `writeRules` (function_declaration) line 125
- `writeMap` (function_declaration) line 146
- `writeManifest` (function_declaration) line 171
- `WriteBundle` (function_declaration) line 191

## internal/gitindex/git.go

- `Commit` (type_spec) line 13
- `BlameInfo` (type_spec) line 22
- `IsGitRepo` (function_declaration) line 28
- `ListCommits` (function_declaration) line 34
- `parseGitLog` (function_declaration) line 55
- `CommitChunks` (function_declaration) line 93
- `BlameFile` (function_declaration) line 112
- `parseBlame` (function_declaration) line 124
- `parseBlameSHA` (function_declaration) line 148

## internal/gitindex/git_test.go

- `TestParseGitLogMultipleCommitsAndFiles` (function_declaration) line 9
- `TestParseBlameExtractsCommitAndAuthor` (function_declaration) line 46
- `TestParseBlameSHA` (function_declaration) line 78
- `TestListCommitsThisRepo` (function_declaration) line 88

## internal/gitindex/ignore.go

- `Ignore` (type_spec) line 13
- `ignoreSource` (type_spec) line 17
- `ignorePat` (type_spec) line 22
- `LoadGitignore` (function_declaration) line 29
- `ParseGitignore` (function_declaration) line 68
- `error` (method_declaration) line 84
- `bool` (method_declaration) line 115
- `bool` (method_declaration) line 139
- `pathInSource` (function_declaration) line 153
- `parseIgnoreLine` (function_declaration) line 164
- `compileIgnoreGlob` (function_declaration) line 196
- `writeGlob` (function_declaration) line 209

## internal/gitindex/ignore_test.go

- `TestParseGitignorePatterns` (function_declaration) line 9
- `TestLoadGitignoreNested` (function_declaration) line 43
- `TestLoadGitignoreMissing` (function_declaration) line 69

## internal/glob/glob.go

- `MatchAnyPattern` (function_declaration) line 11
- `matchDoubleStar` (function_declaration) line 31
- `matchBaseGlob` (function_declaration) line 57

## internal/index/indexer.go

- `Indexer` (type_spec) line 22
- `error` (method_declaration) line 40
- `bool` (method_declaration) line 120
- `bool` (method_declaration) line 124
- `error` (method_declaration) line 134
- `error` (method_declaration) line 200
- `string` (method_declaration) line 232
- `error` (method_declaration) line 239
- `error` (method_declaration) line 292
- `bool` (method_declaration) line 315
- `bool` (method_declaration) line 333
- `bool` (method_declaration) line 348
- `bool` (method_declaration) line 355
- `error` (method_declaration) line 359
- `isArchivePath` (function_declaration) line 372
- `error` (method_declaration) line 384
- `error` (method_declaration) line 404
- `pathInScope` (function_declaration) line 425
- `fileHash` (function_declaration) line 434

## internal/index/indexer_test.go

- `writeFile` (function_declaration) line 14
- `newIndexer` (function_declaration) line 25
- `TestIndexCodeMap` (function_declaration) line 48
- `TestIndexRecord` (function_declaration) line 64
- `TestIndexSkipsArchive` (function_declaration) line 95
- `TestFormatSummary` (function_declaration) line 108

## internal/index/progress.go

- `Phase` (type_spec) line 6
- `string` (method_declaration) line 16
- `Progress` (type_spec) line 34
- `Reporter` (type_spec) line 53
- `FormatSummary` (function_declaration) line 56
- `plural` (function_declaration) line 66

## internal/mcp/server.go

- `Server` (type_spec) line 21
- `New` (function_declaration) line 31
- `jsonResult` (function_declaration) line 94
- `ServeStdio` (function_declaration) line 217
- `ServeHTTP` (function_declaration) line 221

## internal/migrate/migrate.go

- `Records` (function_declaration) line 18
- `migrateDir` (function_declaration) line 40
- `legacyToRecord` (function_declaration) line 79

## internal/publish/publish.go

- `Publish` (function_declaration) line 15

## internal/record/frontmatter.go

- `ParseFile` (function_declaration) line 9
- `Serialize` (function_declaration) line 77
- `splitFrontMatter` (function_declaration) line 111
- `parseYAMLMap` (function_declaration) line 136
- `splitYAMLLine` (function_declaration) line 152
- `unquoteYAML` (function_declaration) line 162
- `yamlQuote` (function_declaration) line 171
- `parseInlineList` (function_declaration) line 178
- `formatInlineList` (function_declaration) line 202
- `titleFromBody` (function_declaration) line 213

## internal/record/record.go

- `Type` (type_spec) line 16
- `Scope` (type_spec) line 26
- `Status` (type_spec) line 34
- `Severity` (type_spec) line 43
- `Record` (type_spec) line 53
- `ScopePrecedence` (function_declaration) line 74
- `InferScopeFromPath` (function_declaration) line 88
- `SlugFromPath` (function_declaration) line 103
- `NewID` (function_declaration) line 113
- `ContentHash` (function_declaration) line 120
- `error` (method_declaration) line 137
- `string` (method_declaration) line 182
- `bool` (method_declaration) line 202
- `bool` (method_declaration) line 215

## internal/record/record_test.go

- `TestParseAndSerialize` (function_declaration) line 8
- `TestMatchesPaths` (function_declaration) line 45

## internal/retrieve/retrieve.go

- `Options` (type_spec) line 17
- `Result` (type_spec) line 24
- `Engine` (type_spec) line 30
- `matchFilters` (function_declaration) line 159
- `rrfScore` (function_declaration) line 169
- `FormatResults` (function_declaration) line 181

## internal/skills/install.go

- `Target` (type_spec) line 10
- `ParseTarget` (function_declaration) line 19
- `Install` (function_declaration) line 34
- `destPath` (function_declaration) line 64
- `wrapCursorMDC` (function_declaration) line 79

## internal/store/embedding.go

- `encodeEmbedding` (function_declaration) line 9
- `decodeEmbedding` (function_declaration) line 21
- `CosineSimilarity` (function_declaration) line 39

## internal/store/store.go

- `SchemaError` (type_spec) line 27
- `string` (method_declaration) line 32
- `Store` (type_spec) line 36
- `Open` (function_declaration) line 40
- `OpenIfExists` (function_declaration) line 56
- `error` (method_declaration) line 74
- `error` (method_declaration) line 78
- `error` (method_declaration) line 173
- `error` (method_declaration) line 183
- `error` (method_declaration) line 206
- `error` (method_declaration) line 308
- `error` (method_declaration) line 340
- `error` (method_declaration) line 347
- `encodeJSONList` (function_declaration) line 376
- `decodeJSONList` (function_declaration) line 384
- `scanRecord` (function_declaration) line 395
- `scanner` (type_spec) line 425
- `error` (method_declaration) line 431
- `nullString` (function_declaration) line 499
- `error` (method_declaration) line 576
- `error` (method_declaration) line 600
- `VectorRecord` (type_spec) line 632
- `QueueItem` (type_spec) line 680
- `error` (method_declaration) line 716
- `FTSResult` (type_spec) line 725
- `FileRecord` (type_spec) line 759
- `Symbol` (type_spec) line 766
- `SymbolEdge` (type_spec) line 776
- `error` (method_declaration) line 797
- `boolToInt` (function_declaration) line 837
- `error` (method_declaration) line 844
- `CommitRecord` (type_spec) line 928
- `error` (method_declaration) line 954

## internal/store/store_test.go

- `TestRecordRoundTrip` (function_declaration) line 14
- `TestSchemaNewerThanCLI` (function_declaration) line 57
- `TestFileMapRoundTrip` (function_declaration) line 80

## internal/version/version.go

- `String` (function_declaration) line 13

