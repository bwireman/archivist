# Code Map

## cmd/archivist/main.go

- `main` (function_declaration) line 10

## internal/archive/service.go

- `Service` (type_spec) line 15
- `New` (function_declaration) line 22
- `error` (method_declaration) line 62
- `error` (method_declaration) line 82
- `string` (method_declaration) line 127
- `string` (method_declaration) line 146
- `string` (method_declaration) line 158
- `slugify` (function_declaration) line 173

## internal/archive/service_test.go

- `TestRememberGlobalWritesHome` (function_declaration) line 13
- `TestRememberGlobalInRepoWhenConfigured` (function_declaration) line 56

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

- `newExportCmd` (function_declaration) line 11

## internal/cmd/index.go

- `newIndexCmd` (function_declaration) line 12

## internal/cmd/init.go

- `newInitCmd` (function_declaration) line 12
- `configExists` (function_declaration) line 40
- `applyInit` (function_declaration) line 45

## internal/cmd/init_test.go

- `TestApplyInitCreatesDecisionDirs` (function_declaration) line 11
- `TestAppendGitignoreInsertsNewline` (function_declaration) line 34

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
- `resolveHomeStorePath` (function_declaration) line 92
- `openStores` (function_declaration) line 100
- `appendGitignore` (function_declaration) line 113
- `newSearchCmd` (function_declaration) line 140
- `newStatusCmd` (function_declaration) line 194
- `status` (type_spec) line 218

## internal/cmd/skills.go

- `newSkillsCmd` (function_declaration) line 10

## internal/cmd/version_test.go

- `TestVersionFlagAndCommand` (function_declaration) line 11

## internal/codemap/codemap.go

- `languageSpec` (type_spec) line 26
- `Result` (type_spec) line 57
- `Extract` (function_declaration) line 66
- `withGenericBackup` (function_declaration) line 79
- `extractTreeSitter` (function_declaration) line 94
- `findPackage` (function_declaration) line 154
- `symbolName` (function_declaration) line 177
- `isExported` (function_declaration) line 191
- `firstDocLine` (function_declaration) line 199
- `cleanImport` (function_declaration) line 219
- `IsBinary` (function_declaration) line 228

## internal/codemap/codemap_test.go

- `TestExtractGleamTopLevel` (function_declaration) line 9
- `keys` (function_declaration) line 92
- `TestExtractGleamConstructors` (function_declaration) line 100
- `TestExtractMjsUsesJavaScript` (function_declaration) line 133
- `TestExtractGenericUnknownLanguage` (function_declaration) line 153
- `TestExtractGenericUnsupportedLanguages` (function_declaration) line 170
- `TestExtractGenericSkipsDocsAndData` (function_declaration) line 231
- `TestExtractGenericBackupWhenTreeSitterFindsNothing` (function_declaration) line 244
- `TestExtractGenericSkipsEnglishStopwords` (function_declaration) line 255

## internal/codemap/generic.go

- `extractGeneric` (function_declaration) line 39
- `isGenericComment` (function_declaration) line 76
- `genericKind` (function_declaration) line 84
- `genericExported` (function_declaration) line 93

## internal/codemap/gleam.go

- `extractGleam` (function_declaration) line 20
- `gleamSym` (function_declaration) line 86

## internal/codemap/languages_test.go

- `TestExtractEveryRegisteredLanguage` (function_declaration) line 26
- `symbolNames` (function_declaration) line 152

## internal/config/config.go

- `Config` (type_spec) line 28
- `PublishConfig` (type_spec) line 35
- `PublishDestination` (type_spec) line 39
- `OllamaConfig` (type_spec) line 43
- `string` (method_declaration) line 54
- `normalizeOllamaURL` (function_declaration) line 64
- `parseTimeout` (function_declaration) line 76
- `IndexConfig` (type_spec) line 87
- `RecordsConfig` (type_spec) line 91
- `RecordsConfig` (method_declaration) line 98
- `expandHomePath` (function_declaration) line 116
- `bool` (method_declaration) line 133
- `string` (method_declaration) line 146
- `PathUnder` (function_declaration) line 163
- `Default` (function_declaration) line 172
- `Load` (function_declaration) line 186
- `Save` (function_declaration) line 212
- `DataDir` (function_declaration) line 222
- `StorePath` (function_declaration) line 226
- `ArchivistHome` (function_declaration) line 230
- `HomeStorePath` (function_declaration) line 238
- `UserRecordsDir` (function_declaration) line 246
- `UserDecisionsDir` (function_declaration) line 254
- `string` (method_declaration) line 258
- `VirtualUserADRPath` (function_declaration) line 274
- `VirtualHomeGlobalPath` (function_declaration) line 278
- `prefixedHomePath` (function_declaration) line 282
- `IsUserGlobalPath` (function_declaration) line 290
- `IsHomeGlobalPath` (function_declaration) line 294
- `hasHomePrefix` (function_declaration) line 298

## internal/config/config_test.go

- `fullConfig` (function_declaration) line 13
- `TestDefaultConfig` (function_declaration) line 37
- `TestLoadFullConfigFromDisk` (function_declaration) line 65
- `TestSaveLoadRoundTrip` (function_declaration) line 84
- `TestLoadMissingConfigUsesDefaults` (function_declaration) line 112
- `TestLoadInvalidJSON` (function_declaration) line 123
- `TestLoadOmitsRecordsUsesDefaults` (function_declaration) line 133
- `TestPathUnder` (function_declaration) line 154
- `TestVirtualUserADRPath` (function_declaration) line 163
- `TestIsUserGlobalPath` (function_declaration) line 169
- `TestIsHomeGlobalPath` (function_declaration) line 178
- `TestGlobalDirDefaultsToArchivistHome` (function_declaration) line 190
- `TestStorePaths` (function_declaration) line 218
- `TestResolvedBaseURLEnv` (function_declaration) line 234
- `TestDataDir` (function_declaration) line 246
- `TestNormalizeOllamaHostWithoutScheme` (function_declaration) line 253

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

- `Indexer` (type_spec) line 23
- `error` (method_declaration) line 42
- `bool` (method_declaration) line 136
- `bool` (method_declaration) line 146
- `error` (method_declaration) line 159
- `error` (method_declaration) line 232
- `string` (method_declaration) line 264
- `error` (method_declaration) line 274
- `error` (method_declaration) line 303
- `samePath` (function_declaration) line 385
- `error` (method_declaration) line 403
- `bool` (method_declaration) line 426
- `bool` (method_declaration) line 442
- `bool` (method_declaration) line 460
- `bool` (method_declaration) line 467
- `error` (method_declaration) line 471
- `bool` (method_declaration) line 499
- `error` (method_declaration) line 514
- `error` (method_declaration) line 534
- `pathInScope` (function_declaration) line 555
- `fileHash` (function_declaration) line 564

## internal/index/indexer_test.go

- `writeFile` (function_declaration) line 14
- `newIndexer` (function_declaration) line 25
- `TestIndexCodeMap` (function_declaration) line 49
- `TestIndexGleamCodeMap` (function_declaration) line 65
- `TestIndexRemapsWhenCodemapVersionStale` (function_declaration) line 81
- `TestIndexSkipsUnchangedWhenCodemapCurrent` (function_declaration) line 110
- `TestIndexRecord` (function_declaration) line 136
- `TestIndexHomeGlobalRecord` (function_declaration) line 167
- `TestIndexSkipsArchive` (function_declaration) line 211
- `TestFormatSummary` (function_declaration) line 224

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
- `ValidType` (function_declaration) line 31
- `Scope` (type_spec) line 40
- `Status` (type_spec) line 48
- `Severity` (type_spec) line 57
- `Record` (type_spec) line 67
- `ScopePrecedence` (function_declaration) line 88
- `InferScopeFromPath` (function_declaration) line 102
- `SlugFromPath` (function_declaration) line 120
- `NewID` (function_declaration) line 130
- `ContentHash` (function_declaration) line 137
- `error` (method_declaration) line 154
- `string` (method_declaration) line 197
- `bool` (method_declaration) line 217
- `bool` (method_declaration) line 230

## internal/record/record_test.go

- `TestParseAndSerialize` (function_declaration) line 8
- `TestMatchesPaths` (function_declaration) line 45
- `TestValidateFeatureType` (function_declaration) line 58
- `TestIndexOrderCoversKnownTypes` (function_declaration) line 69
- `TestInferScopeFromPath` (function_declaration) line 85

## internal/retrieve/retrieve.go

- `Options` (type_spec) line 17
- `Result` (type_spec) line 24
- `Engine` (type_spec) line 30
- `matchFilters` (function_declaration) line 159
- `rrfScore` (function_declaration) line 169
- `FormatResults` (function_declaration) line 181

## internal/skills/install.go

- `Target` (type_spec) line 15
- `ParseTarget` (function_declaration) line 24
- `Install` (function_declaration) line 40
- `installRules` (function_declaration) line 52
- `installSkills` (function_declaration) line 94
- `ruleFS` (function_declaration) line 124
- `skillFS` (function_declaration) line 131
- `dirWithSuffix` (function_declaration) line 148
- `wrapCursorRule` (function_declaration) line 161
- `writeFile` (function_declaration) line 165

## internal/skills/install_test.go

- `TestInstallCursor` (function_declaration) line 10
- `TestInstallCursorEmbeddedTemplates` (function_declaration) line 30
- `TestInstallAgentsMDWritesRulesOnly` (function_declaration) line 53
- `writeTree` (function_declaration) line 71

## internal/store/embedding.go

- `encodeEmbedding` (function_declaration) line 9
- `decodeEmbedding` (function_declaration) line 21
- `CosineSimilarity` (function_declaration) line 39

## internal/store/fts.go

- `fts5Query` (function_declaration) line 13

## internal/store/fts_test.go

- `TestFTS5Query` (function_declaration) line 10
- `TestSearchFTSAcceptsPunctuation` (function_declaration) line 35
- `openFTSStore` (function_declaration) line 81

## internal/store/store.go

- `SchemaError` (type_spec) line 28
- `string` (method_declaration) line 33
- `Store` (type_spec) line 37
- `Open` (function_declaration) line 41
- `OpenIfExists` (function_declaration) line 57
- `error` (method_declaration) line 75
- `error` (method_declaration) line 79
- `error` (method_declaration) line 174
- `error` (method_declaration) line 184
- `error` (method_declaration) line 207
- `error` (method_declaration) line 309
- `error` (method_declaration) line 341
- `error` (method_declaration) line 348
- `encodeJSONList` (function_declaration) line 377
- `decodeJSONList` (function_declaration) line 385
- `scanRecord` (function_declaration) line 396
- `scanner` (type_spec) line 426
- `error` (method_declaration) line 432
- `nullString` (function_declaration) line 500
- `error` (method_declaration) line 577
- `error` (method_declaration) line 601
- `VectorRecord` (type_spec) line 633
- `QueueItem` (type_spec) line 681
- `error` (method_declaration) line 717
- `FTSResult` (type_spec) line 726
- `FileRecord` (type_spec) line 763
- `Symbol` (type_spec) line 770
- `SymbolEdge` (type_spec) line 780
- `error` (method_declaration) line 801
- `boolToInt` (function_declaration) line 841
- `error` (method_declaration) line 848
- `CommitRecord` (type_spec) line 932
- `error` (method_declaration) line 958

## internal/store/store_test.go

- `TestRecordRoundTrip` (function_declaration) line 14
- `TestSchemaNewerThanCLI` (function_declaration) line 57
- `TestFileMapRoundTrip` (function_declaration) line 80

## internal/version/version.go

- `String` (function_declaration) line 13

