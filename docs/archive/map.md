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
- `slugify` (function_declaration) line 162

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

## internal/cmd/embed.go

- `newEmbedCmd` (function_declaration) line 11

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
- `newStatusCmd` (function_declaration) line 190
- `status` (type_spec) line 215

## internal/cmd/skills.go

- `newSkillsCmd` (function_declaration) line 10

## internal/cmd/version_test.go

- `TestVersionFlagAndCommand` (function_declaration) line 11

## internal/codemap/codemap.go

- `languageSpec` (type_spec) line 27
- `Result` (type_spec) line 58
- `Extract` (function_declaration) line 67
- `withGenericBackup` (function_declaration) line 80
- `parserFor` (function_declaration) line 100
- `extractTreeSitter` (function_declaration) line 112
- `findPackage` (function_declaration) line 171
- `symbolName` (function_declaration) line 194
- `isExported` (function_declaration) line 208
- `firstDocLine` (function_declaration) line 216
- `cleanImport` (function_declaration) line 236
- `IsBinary` (function_declaration) line 245

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
- `string` (method_declaration) line 254
- `VirtualUserADRPath` (function_declaration) line 270
- `VirtualHomeGlobalPath` (function_declaration) line 274
- `prefixedHomePath` (function_declaration) line 278
- `IsUserGlobalPath` (function_declaration) line 286
- `IsHomeGlobalPath` (function_declaration) line 290
- `hasHomePrefix` (function_declaration) line 294

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
- `OptionalFromConfig` (function_declaration) line 14
- `HealthStatus` (type_spec) line 22
- `CheckHealth` (function_declaration) line 27

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

- `Worker` (type_spec) line 15
- `WorkerOptions` (type_spec) line 21
- `int` (method_declaration) line 53
- `job` (type_spec) line 68

## internal/embed/worker_test.go

- `TestWorkerOnceDrainsWholeQueue` (function_declaration) line 16
- `TestWorkerContinuesAfterOneFailure` (function_declaration) line 36
- `TestWorkerDropsGhostQueueItems` (function_declaration) line 66
- `TestWorkerEmbedsHomeStoreWhenRepoIsEmpty` (function_declaration) line 78
- `TestWorkerFindsRecordOnOtherStore` (function_declaration) line 96
- `TestWorkerWithoutOnceDrainsQueue` (function_declaration) line 114
- `embedFunc` (type_spec) line 133
- `int` (method_declaration) line 139
- `openStore` (function_declaration) line 141
- `upsertQueued` (function_declaration) line 151

## internal/export/export.go

- `Options` (type_spec) line 19
- `Manifest` (type_spec) line 24
- `Entry` (type_spec) line 30
- `Run` (function_declaration) line 44
- `collectRecords` (function_declaration) line 70
- `writeRecords` (function_declaration) line 91
- `writeIndex` (function_declaration) line 107
- `writeTypeDigests` (function_declaration) line 141
- `writeTypeDigest` (function_declaration) line 159
- `groupByType` (function_declaration) line 182
- `recordRel` (function_declaration) line 190
- `wantDigest` (function_declaration) line 194
- `typeHeading` (function_declaration) line 198
- `digestFile` (function_declaration) line 205
- `writeMap` (function_declaration) line 209
- `writeManifest` (function_declaration) line 236
- `WriteBundle` (function_declaration) line 256

## internal/export/export_test.go

- `TestRunNestsRecordsScopeThenType` (function_declaration) line 14
- `TestRunWritesCodeMapFromSymbols` (function_declaration) line 102
- `TestRunWritesMapsDigestBesideCodeMap` (function_declaration) line 136
- `TestRunAlwaysWritesRulesDigest` (function_declaration) line 163
- `openStore` (function_declaration) line 178
- `mustUpsert` (function_declaration) line 188
- `mustNow` (function_declaration) line 195

## internal/gitindex/git.go

- `Commit` (type_spec) line 10
- `IsGitRepo` (function_declaration) line 19
- `ListCommits` (function_declaration) line 25
- `parseGitLog` (function_declaration) line 43

## internal/gitindex/git_test.go

- `TestParseGitLogMultipleCommits` (function_declaration) line 8
- `TestListCommitsThisRepo` (function_declaration) line 38

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

## internal/glob/glob_test.go

- `TestMatchAnyPatternSkipGlobs` (function_declaration) line 5

## internal/index/indexer.go

- `Indexer` (type_spec) line 23
- `error` (method_declaration) line 42
- `bool` (method_declaration) line 128
- `bool` (method_declaration) line 138
- `error` (method_declaration) line 151
- `error` (method_declaration) line 221
- `string` (method_declaration) line 255
- `error` (method_declaration) line 265
- `error` (method_declaration) line 294
- `samePath` (function_declaration) line 333
- `error` (method_declaration) line 342
- `bool` (method_declaration) line 365
- `bool` (method_declaration) line 381
- `bool` (method_declaration) line 399
- `bool` (method_declaration) line 406
- `error` (method_declaration) line 410
- `bool` (method_declaration) line 438
- `error` (method_declaration) line 453
- `error` (method_declaration) line 473
- `pathInScope` (function_declaration) line 494
- `fileHash` (function_declaration) line 503

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
- `TestIndexInRepoGlobalRecordNotPruned` (function_declaration) line 224
- `TestFormatSummary` (function_declaration) line 276

## internal/index/progress.go

- `Phase` (type_spec) line 6
- `string` (method_declaration) line 16
- `Progress` (type_spec) line 34
- `Reporter` (type_spec) line 44
- `FormatSummary` (function_declaration) line 47
- `plural` (function_declaration) line 57

## internal/mcp/server.go

- `Server` (type_spec) line 20
- `New` (function_declaration) line 30
- `jsonResult` (function_declaration) line 93
- `ServeStdio` (function_declaration) line 222

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
- `TitleFromBody` (function_declaration) line 213

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

- `Options` (type_spec) line 18
- `Result` (type_spec) line 25
- `Engine` (type_spec) line 31
- `rrfScore` (function_declaration) line 172
- `FormatResults` (function_declaration) line 184

## internal/retrieve/retrieve_test.go

- `fixedEmbedder` (type_spec) line 13
- `int` (method_declaration) line 21
- `TestSearchFiltersByType` (function_declaration) line 23
- `TestSearchOverlayPrefersRepoScope` (function_declaration) line 67

## internal/skills/install.go

- `Target` (type_spec) line 16
- `ParseTarget` (function_declaration) line 25
- `Install` (function_declaration) line 41
- `installRules` (function_declaration) line 53
- `installSkills` (function_declaration) line 95
- `ruleFS` (function_declaration) line 128
- `skillFS` (function_declaration) line 135
- `dirWithSuffix` (function_declaration) line 152
- `wrapCursorRule` (function_declaration) line 165
- `writeFile` (function_declaration) line 169

## internal/skills/install_test.go

- `TestInstallCursor` (function_declaration) line 10
- `TestInstallCursorEmbeddedTemplates` (function_declaration) line 30
- `TestInstallAgentsMDWritesRulesOnly` (function_declaration) line 53
- `writeTree` (function_declaration) line 71

## internal/store/embedding.go

- `encodeEmbedding` (function_declaration) line 9
- `decodeEmbedding` (function_declaration) line 21
- `CosineSimilarity` (function_declaration) line 39

## internal/store/embedding_test.go

- `TestListEmbeddingsFiltersType` (function_declaration) line 10

## internal/store/fts.go

- `fts5Query` (function_declaration) line 13

## internal/store/fts_test.go

- `TestFTS5Query` (function_declaration) line 10
- `TestSearchFTSAcceptsPunctuation` (function_declaration) line 35
- `openFTSStore` (function_declaration) line 81

## internal/store/store.go

- `SchemaError` (type_spec) line 29
- `string` (method_declaration) line 34
- `Store` (type_spec) line 38
- `Open` (function_declaration) line 42
- `OpenIfExists` (function_declaration) line 66
- `error` (method_declaration) line 84
- `error` (method_declaration) line 88
- `error` (method_declaration) line 183
- `error` (method_declaration) line 191
- `error` (method_declaration) line 201
- `error` (method_declaration) line 224
- `error` (method_declaration) line 326
- `error` (method_declaration) line 358
- `error` (method_declaration) line 365
- `encodeJSONList` (function_declaration) line 394
- `decodeJSONList` (function_declaration) line 402
- `scanRecord` (function_declaration) line 413
- `scanner` (type_spec) line 443
- `error` (method_declaration) line 449
- `nullString` (function_declaration) line 517
- `error` (method_declaration) line 611
- `error` (method_declaration) line 643
- `RecordFilter` (type_spec) line 676
- `EmbeddingRow` (type_spec) line 681
- `QueueItem` (type_spec) line 723
- `error` (method_declaration) line 765
- `error` (method_declaration) line 770
- `FTSResult` (type_spec) line 779
- `FileRecord` (type_spec) line 826
- `Symbol` (type_spec) line 833
- `SymbolEdge` (type_spec) line 843
- `error` (method_declaration) line 864
- `boolToInt` (function_declaration) line 919
- `error` (method_declaration) line 926
- `collectSymbols` (function_declaration) line 996
- `CommitRecord` (type_spec) line 1013
- `error` (method_declaration) line 1039

## internal/store/store_test.go

- `TestRecordRoundTrip` (function_declaration) line 18
- `TestSchemaNewerThanCLI` (function_declaration) line 61
- `TestFileMapRoundTrip` (function_declaration) line 84
- `TestDeleteRecordByPathDropsQueue` (function_declaration) line 110
- `TestDequeueEmbedZeroLimitReturnsAll` (function_declaration) line 141
- `TestOpenPurgesOrphanQueueAndVectors` (function_declaration) line 178

## internal/version/version.go

- `String` (function_declaration) line 13

