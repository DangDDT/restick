// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'app_database.dart';

// ignore_for_file: type=lint
class $LocalPacksTable extends LocalPacks
    with TableInfo<$LocalPacksTable, LocalPack> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $LocalPacksTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _slugMeta = const VerificationMeta('slug');
  @override
  late final GeneratedColumn<String> slug = GeneratedColumn<String>(
    'slug',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _nameMeta = const VerificationMeta('name');
  @override
  late final GeneratedColumn<String> name = GeneratedColumn<String>(
    'name',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _authorMeta = const VerificationMeta('author');
  @override
  late final GeneratedColumn<String> author = GeneratedColumn<String>(
    'author',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _tagsJsonMeta = const VerificationMeta(
    'tagsJson',
  );
  @override
  late final GeneratedColumn<String> tagsJson = GeneratedColumn<String>(
    'tags_json',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _stickerCountMeta = const VerificationMeta(
    'stickerCount',
  );
  @override
  late final GeneratedColumn<int> stickerCount = GeneratedColumn<int>(
    'sticker_count',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _versionMeta = const VerificationMeta(
    'version',
  );
  @override
  late final GeneratedColumn<int> version = GeneratedColumn<int>(
    'version',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _thumbnailUrlMeta = const VerificationMeta(
    'thumbnailUrl',
  );
  @override
  late final GeneratedColumn<String> thumbnailUrl = GeneratedColumn<String>(
    'thumbnail_url',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _checksumMeta = const VerificationMeta(
    'checksum',
  );
  @override
  late final GeneratedColumn<String> checksum = GeneratedColumn<String>(
    'checksum',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _statusMeta = const VerificationMeta('status');
  @override
  late final GeneratedColumn<String> status = GeneratedColumn<String>(
    'status',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _localDirMeta = const VerificationMeta(
    'localDir',
  );
  @override
  late final GeneratedColumn<String> localDir = GeneratedColumn<String>(
    'local_dir',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _updatedAtMeta = const VerificationMeta(
    'updatedAt',
  );
  @override
  late final GeneratedColumn<DateTime> updatedAt = GeneratedColumn<DateTime>(
    'updated_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    slug,
    name,
    author,
    tagsJson,
    stickerCount,
    version,
    thumbnailUrl,
    checksum,
    status,
    localDir,
    updatedAt,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'local_packs';
  @override
  VerificationContext validateIntegrity(
    Insertable<LocalPack> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('slug')) {
      context.handle(
        _slugMeta,
        slug.isAcceptableOrUnknown(data['slug']!, _slugMeta),
      );
    } else if (isInserting) {
      context.missing(_slugMeta);
    }
    if (data.containsKey('name')) {
      context.handle(
        _nameMeta,
        name.isAcceptableOrUnknown(data['name']!, _nameMeta),
      );
    } else if (isInserting) {
      context.missing(_nameMeta);
    }
    if (data.containsKey('author')) {
      context.handle(
        _authorMeta,
        author.isAcceptableOrUnknown(data['author']!, _authorMeta),
      );
    } else if (isInserting) {
      context.missing(_authorMeta);
    }
    if (data.containsKey('tags_json')) {
      context.handle(
        _tagsJsonMeta,
        tagsJson.isAcceptableOrUnknown(data['tags_json']!, _tagsJsonMeta),
      );
    } else if (isInserting) {
      context.missing(_tagsJsonMeta);
    }
    if (data.containsKey('sticker_count')) {
      context.handle(
        _stickerCountMeta,
        stickerCount.isAcceptableOrUnknown(
          data['sticker_count']!,
          _stickerCountMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_stickerCountMeta);
    }
    if (data.containsKey('version')) {
      context.handle(
        _versionMeta,
        version.isAcceptableOrUnknown(data['version']!, _versionMeta),
      );
    } else if (isInserting) {
      context.missing(_versionMeta);
    }
    if (data.containsKey('thumbnail_url')) {
      context.handle(
        _thumbnailUrlMeta,
        thumbnailUrl.isAcceptableOrUnknown(
          data['thumbnail_url']!,
          _thumbnailUrlMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_thumbnailUrlMeta);
    }
    if (data.containsKey('checksum')) {
      context.handle(
        _checksumMeta,
        checksum.isAcceptableOrUnknown(data['checksum']!, _checksumMeta),
      );
    } else if (isInserting) {
      context.missing(_checksumMeta);
    }
    if (data.containsKey('status')) {
      context.handle(
        _statusMeta,
        status.isAcceptableOrUnknown(data['status']!, _statusMeta),
      );
    } else if (isInserting) {
      context.missing(_statusMeta);
    }
    if (data.containsKey('local_dir')) {
      context.handle(
        _localDirMeta,
        localDir.isAcceptableOrUnknown(data['local_dir']!, _localDirMeta),
      );
    } else if (isInserting) {
      context.missing(_localDirMeta);
    }
    if (data.containsKey('updated_at')) {
      context.handle(
        _updatedAtMeta,
        updatedAt.isAcceptableOrUnknown(data['updated_at']!, _updatedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_updatedAtMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  LocalPack map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return LocalPack(
      id:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}id'],
          )!,
      slug:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}slug'],
          )!,
      name:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}name'],
          )!,
      author:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}author'],
          )!,
      tagsJson:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}tags_json'],
          )!,
      stickerCount:
          attachedDatabase.typeMapping.read(
            DriftSqlType.int,
            data['${effectivePrefix}sticker_count'],
          )!,
      version:
          attachedDatabase.typeMapping.read(
            DriftSqlType.int,
            data['${effectivePrefix}version'],
          )!,
      thumbnailUrl:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}thumbnail_url'],
          )!,
      checksum:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}checksum'],
          )!,
      status:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}status'],
          )!,
      localDir:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}local_dir'],
          )!,
      updatedAt:
          attachedDatabase.typeMapping.read(
            DriftSqlType.dateTime,
            data['${effectivePrefix}updated_at'],
          )!,
    );
  }

  @override
  $LocalPacksTable createAlias(String alias) {
    return $LocalPacksTable(attachedDatabase, alias);
  }
}

class LocalPack extends DataClass implements Insertable<LocalPack> {
  final String id;
  final String slug;
  final String name;
  final String author;
  final String tagsJson;
  final int stickerCount;
  final int version;
  final String thumbnailUrl;
  final String checksum;
  final String status;
  final String localDir;
  final DateTime updatedAt;
  const LocalPack({
    required this.id,
    required this.slug,
    required this.name,
    required this.author,
    required this.tagsJson,
    required this.stickerCount,
    required this.version,
    required this.thumbnailUrl,
    required this.checksum,
    required this.status,
    required this.localDir,
    required this.updatedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['slug'] = Variable<String>(slug);
    map['name'] = Variable<String>(name);
    map['author'] = Variable<String>(author);
    map['tags_json'] = Variable<String>(tagsJson);
    map['sticker_count'] = Variable<int>(stickerCount);
    map['version'] = Variable<int>(version);
    map['thumbnail_url'] = Variable<String>(thumbnailUrl);
    map['checksum'] = Variable<String>(checksum);
    map['status'] = Variable<String>(status);
    map['local_dir'] = Variable<String>(localDir);
    map['updated_at'] = Variable<DateTime>(updatedAt);
    return map;
  }

  LocalPacksCompanion toCompanion(bool nullToAbsent) {
    return LocalPacksCompanion(
      id: Value(id),
      slug: Value(slug),
      name: Value(name),
      author: Value(author),
      tagsJson: Value(tagsJson),
      stickerCount: Value(stickerCount),
      version: Value(version),
      thumbnailUrl: Value(thumbnailUrl),
      checksum: Value(checksum),
      status: Value(status),
      localDir: Value(localDir),
      updatedAt: Value(updatedAt),
    );
  }

  factory LocalPack.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return LocalPack(
      id: serializer.fromJson<String>(json['id']),
      slug: serializer.fromJson<String>(json['slug']),
      name: serializer.fromJson<String>(json['name']),
      author: serializer.fromJson<String>(json['author']),
      tagsJson: serializer.fromJson<String>(json['tagsJson']),
      stickerCount: serializer.fromJson<int>(json['stickerCount']),
      version: serializer.fromJson<int>(json['version']),
      thumbnailUrl: serializer.fromJson<String>(json['thumbnailUrl']),
      checksum: serializer.fromJson<String>(json['checksum']),
      status: serializer.fromJson<String>(json['status']),
      localDir: serializer.fromJson<String>(json['localDir']),
      updatedAt: serializer.fromJson<DateTime>(json['updatedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'slug': serializer.toJson<String>(slug),
      'name': serializer.toJson<String>(name),
      'author': serializer.toJson<String>(author),
      'tagsJson': serializer.toJson<String>(tagsJson),
      'stickerCount': serializer.toJson<int>(stickerCount),
      'version': serializer.toJson<int>(version),
      'thumbnailUrl': serializer.toJson<String>(thumbnailUrl),
      'checksum': serializer.toJson<String>(checksum),
      'status': serializer.toJson<String>(status),
      'localDir': serializer.toJson<String>(localDir),
      'updatedAt': serializer.toJson<DateTime>(updatedAt),
    };
  }

  LocalPack copyWith({
    String? id,
    String? slug,
    String? name,
    String? author,
    String? tagsJson,
    int? stickerCount,
    int? version,
    String? thumbnailUrl,
    String? checksum,
    String? status,
    String? localDir,
    DateTime? updatedAt,
  }) => LocalPack(
    id: id ?? this.id,
    slug: slug ?? this.slug,
    name: name ?? this.name,
    author: author ?? this.author,
    tagsJson: tagsJson ?? this.tagsJson,
    stickerCount: stickerCount ?? this.stickerCount,
    version: version ?? this.version,
    thumbnailUrl: thumbnailUrl ?? this.thumbnailUrl,
    checksum: checksum ?? this.checksum,
    status: status ?? this.status,
    localDir: localDir ?? this.localDir,
    updatedAt: updatedAt ?? this.updatedAt,
  );
  LocalPack copyWithCompanion(LocalPacksCompanion data) {
    return LocalPack(
      id: data.id.present ? data.id.value : this.id,
      slug: data.slug.present ? data.slug.value : this.slug,
      name: data.name.present ? data.name.value : this.name,
      author: data.author.present ? data.author.value : this.author,
      tagsJson: data.tagsJson.present ? data.tagsJson.value : this.tagsJson,
      stickerCount:
          data.stickerCount.present
              ? data.stickerCount.value
              : this.stickerCount,
      version: data.version.present ? data.version.value : this.version,
      thumbnailUrl:
          data.thumbnailUrl.present
              ? data.thumbnailUrl.value
              : this.thumbnailUrl,
      checksum: data.checksum.present ? data.checksum.value : this.checksum,
      status: data.status.present ? data.status.value : this.status,
      localDir: data.localDir.present ? data.localDir.value : this.localDir,
      updatedAt: data.updatedAt.present ? data.updatedAt.value : this.updatedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('LocalPack(')
          ..write('id: $id, ')
          ..write('slug: $slug, ')
          ..write('name: $name, ')
          ..write('author: $author, ')
          ..write('tagsJson: $tagsJson, ')
          ..write('stickerCount: $stickerCount, ')
          ..write('version: $version, ')
          ..write('thumbnailUrl: $thumbnailUrl, ')
          ..write('checksum: $checksum, ')
          ..write('status: $status, ')
          ..write('localDir: $localDir, ')
          ..write('updatedAt: $updatedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    id,
    slug,
    name,
    author,
    tagsJson,
    stickerCount,
    version,
    thumbnailUrl,
    checksum,
    status,
    localDir,
    updatedAt,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is LocalPack &&
          other.id == this.id &&
          other.slug == this.slug &&
          other.name == this.name &&
          other.author == this.author &&
          other.tagsJson == this.tagsJson &&
          other.stickerCount == this.stickerCount &&
          other.version == this.version &&
          other.thumbnailUrl == this.thumbnailUrl &&
          other.checksum == this.checksum &&
          other.status == this.status &&
          other.localDir == this.localDir &&
          other.updatedAt == this.updatedAt);
}

class LocalPacksCompanion extends UpdateCompanion<LocalPack> {
  final Value<String> id;
  final Value<String> slug;
  final Value<String> name;
  final Value<String> author;
  final Value<String> tagsJson;
  final Value<int> stickerCount;
  final Value<int> version;
  final Value<String> thumbnailUrl;
  final Value<String> checksum;
  final Value<String> status;
  final Value<String> localDir;
  final Value<DateTime> updatedAt;
  final Value<int> rowid;
  const LocalPacksCompanion({
    this.id = const Value.absent(),
    this.slug = const Value.absent(),
    this.name = const Value.absent(),
    this.author = const Value.absent(),
    this.tagsJson = const Value.absent(),
    this.stickerCount = const Value.absent(),
    this.version = const Value.absent(),
    this.thumbnailUrl = const Value.absent(),
    this.checksum = const Value.absent(),
    this.status = const Value.absent(),
    this.localDir = const Value.absent(),
    this.updatedAt = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  LocalPacksCompanion.insert({
    required String id,
    required String slug,
    required String name,
    required String author,
    required String tagsJson,
    required int stickerCount,
    required int version,
    required String thumbnailUrl,
    required String checksum,
    required String status,
    required String localDir,
    required DateTime updatedAt,
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       slug = Value(slug),
       name = Value(name),
       author = Value(author),
       tagsJson = Value(tagsJson),
       stickerCount = Value(stickerCount),
       version = Value(version),
       thumbnailUrl = Value(thumbnailUrl),
       checksum = Value(checksum),
       status = Value(status),
       localDir = Value(localDir),
       updatedAt = Value(updatedAt);
  static Insertable<LocalPack> custom({
    Expression<String>? id,
    Expression<String>? slug,
    Expression<String>? name,
    Expression<String>? author,
    Expression<String>? tagsJson,
    Expression<int>? stickerCount,
    Expression<int>? version,
    Expression<String>? thumbnailUrl,
    Expression<String>? checksum,
    Expression<String>? status,
    Expression<String>? localDir,
    Expression<DateTime>? updatedAt,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (slug != null) 'slug': slug,
      if (name != null) 'name': name,
      if (author != null) 'author': author,
      if (tagsJson != null) 'tags_json': tagsJson,
      if (stickerCount != null) 'sticker_count': stickerCount,
      if (version != null) 'version': version,
      if (thumbnailUrl != null) 'thumbnail_url': thumbnailUrl,
      if (checksum != null) 'checksum': checksum,
      if (status != null) 'status': status,
      if (localDir != null) 'local_dir': localDir,
      if (updatedAt != null) 'updated_at': updatedAt,
      if (rowid != null) 'rowid': rowid,
    });
  }

  LocalPacksCompanion copyWith({
    Value<String>? id,
    Value<String>? slug,
    Value<String>? name,
    Value<String>? author,
    Value<String>? tagsJson,
    Value<int>? stickerCount,
    Value<int>? version,
    Value<String>? thumbnailUrl,
    Value<String>? checksum,
    Value<String>? status,
    Value<String>? localDir,
    Value<DateTime>? updatedAt,
    Value<int>? rowid,
  }) {
    return LocalPacksCompanion(
      id: id ?? this.id,
      slug: slug ?? this.slug,
      name: name ?? this.name,
      author: author ?? this.author,
      tagsJson: tagsJson ?? this.tagsJson,
      stickerCount: stickerCount ?? this.stickerCount,
      version: version ?? this.version,
      thumbnailUrl: thumbnailUrl ?? this.thumbnailUrl,
      checksum: checksum ?? this.checksum,
      status: status ?? this.status,
      localDir: localDir ?? this.localDir,
      updatedAt: updatedAt ?? this.updatedAt,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (slug.present) {
      map['slug'] = Variable<String>(slug.value);
    }
    if (name.present) {
      map['name'] = Variable<String>(name.value);
    }
    if (author.present) {
      map['author'] = Variable<String>(author.value);
    }
    if (tagsJson.present) {
      map['tags_json'] = Variable<String>(tagsJson.value);
    }
    if (stickerCount.present) {
      map['sticker_count'] = Variable<int>(stickerCount.value);
    }
    if (version.present) {
      map['version'] = Variable<int>(version.value);
    }
    if (thumbnailUrl.present) {
      map['thumbnail_url'] = Variable<String>(thumbnailUrl.value);
    }
    if (checksum.present) {
      map['checksum'] = Variable<String>(checksum.value);
    }
    if (status.present) {
      map['status'] = Variable<String>(status.value);
    }
    if (localDir.present) {
      map['local_dir'] = Variable<String>(localDir.value);
    }
    if (updatedAt.present) {
      map['updated_at'] = Variable<DateTime>(updatedAt.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('LocalPacksCompanion(')
          ..write('id: $id, ')
          ..write('slug: $slug, ')
          ..write('name: $name, ')
          ..write('author: $author, ')
          ..write('tagsJson: $tagsJson, ')
          ..write('stickerCount: $stickerCount, ')
          ..write('version: $version, ')
          ..write('thumbnailUrl: $thumbnailUrl, ')
          ..write('checksum: $checksum, ')
          ..write('status: $status, ')
          ..write('localDir: $localDir, ')
          ..write('updatedAt: $updatedAt, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $SyncEntriesTable extends SyncEntries
    with TableInfo<$SyncEntriesTable, SyncEntry> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $SyncEntriesTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _keyMeta = const VerificationMeta('key');
  @override
  late final GeneratedColumn<String> key = GeneratedColumn<String>(
    'key',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _valueMeta = const VerificationMeta('value');
  @override
  late final GeneratedColumn<String> value = GeneratedColumn<String>(
    'value',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [key, value];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'sync_entries';
  @override
  VerificationContext validateIntegrity(
    Insertable<SyncEntry> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('key')) {
      context.handle(
        _keyMeta,
        key.isAcceptableOrUnknown(data['key']!, _keyMeta),
      );
    } else if (isInserting) {
      context.missing(_keyMeta);
    }
    if (data.containsKey('value')) {
      context.handle(
        _valueMeta,
        value.isAcceptableOrUnknown(data['value']!, _valueMeta),
      );
    } else if (isInserting) {
      context.missing(_valueMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {key};
  @override
  SyncEntry map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return SyncEntry(
      key:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}key'],
          )!,
      value:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}value'],
          )!,
    );
  }

  @override
  $SyncEntriesTable createAlias(String alias) {
    return $SyncEntriesTable(attachedDatabase, alias);
  }
}

class SyncEntry extends DataClass implements Insertable<SyncEntry> {
  final String key;
  final String value;
  const SyncEntry({required this.key, required this.value});
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['key'] = Variable<String>(key);
    map['value'] = Variable<String>(value);
    return map;
  }

  SyncEntriesCompanion toCompanion(bool nullToAbsent) {
    return SyncEntriesCompanion(key: Value(key), value: Value(value));
  }

  factory SyncEntry.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return SyncEntry(
      key: serializer.fromJson<String>(json['key']),
      value: serializer.fromJson<String>(json['value']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'key': serializer.toJson<String>(key),
      'value': serializer.toJson<String>(value),
    };
  }

  SyncEntry copyWith({String? key, String? value}) =>
      SyncEntry(key: key ?? this.key, value: value ?? this.value);
  SyncEntry copyWithCompanion(SyncEntriesCompanion data) {
    return SyncEntry(
      key: data.key.present ? data.key.value : this.key,
      value: data.value.present ? data.value.value : this.value,
    );
  }

  @override
  String toString() {
    return (StringBuffer('SyncEntry(')
          ..write('key: $key, ')
          ..write('value: $value')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(key, value);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is SyncEntry &&
          other.key == this.key &&
          other.value == this.value);
}

class SyncEntriesCompanion extends UpdateCompanion<SyncEntry> {
  final Value<String> key;
  final Value<String> value;
  final Value<int> rowid;
  const SyncEntriesCompanion({
    this.key = const Value.absent(),
    this.value = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  SyncEntriesCompanion.insert({
    required String key,
    required String value,
    this.rowid = const Value.absent(),
  }) : key = Value(key),
       value = Value(value);
  static Insertable<SyncEntry> custom({
    Expression<String>? key,
    Expression<String>? value,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (key != null) 'key': key,
      if (value != null) 'value': value,
      if (rowid != null) 'rowid': rowid,
    });
  }

  SyncEntriesCompanion copyWith({
    Value<String>? key,
    Value<String>? value,
    Value<int>? rowid,
  }) {
    return SyncEntriesCompanion(
      key: key ?? this.key,
      value: value ?? this.value,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (key.present) {
      map['key'] = Variable<String>(key.value);
    }
    if (value.present) {
      map['value'] = Variable<String>(value.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('SyncEntriesCompanion(')
          ..write('key: $key, ')
          ..write('value: $value, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

abstract class _$RestickDatabase extends GeneratedDatabase {
  _$RestickDatabase(QueryExecutor e) : super(e);
  $RestickDatabaseManager get managers => $RestickDatabaseManager(this);
  late final $LocalPacksTable localPacks = $LocalPacksTable(this);
  late final $SyncEntriesTable syncEntries = $SyncEntriesTable(this);
  @override
  Iterable<TableInfo<Table, Object?>> get allTables =>
      allSchemaEntities.whereType<TableInfo<Table, Object?>>();
  @override
  List<DatabaseSchemaEntity> get allSchemaEntities => [localPacks, syncEntries];
}

typedef $$LocalPacksTableCreateCompanionBuilder =
    LocalPacksCompanion Function({
      required String id,
      required String slug,
      required String name,
      required String author,
      required String tagsJson,
      required int stickerCount,
      required int version,
      required String thumbnailUrl,
      required String checksum,
      required String status,
      required String localDir,
      required DateTime updatedAt,
      Value<int> rowid,
    });
typedef $$LocalPacksTableUpdateCompanionBuilder =
    LocalPacksCompanion Function({
      Value<String> id,
      Value<String> slug,
      Value<String> name,
      Value<String> author,
      Value<String> tagsJson,
      Value<int> stickerCount,
      Value<int> version,
      Value<String> thumbnailUrl,
      Value<String> checksum,
      Value<String> status,
      Value<String> localDir,
      Value<DateTime> updatedAt,
      Value<int> rowid,
    });

class $$LocalPacksTableFilterComposer
    extends Composer<_$RestickDatabase, $LocalPacksTable> {
  $$LocalPacksTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get slug => $composableBuilder(
    column: $table.slug,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get name => $composableBuilder(
    column: $table.name,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get author => $composableBuilder(
    column: $table.author,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get tagsJson => $composableBuilder(
    column: $table.tagsJson,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get stickerCount => $composableBuilder(
    column: $table.stickerCount,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get version => $composableBuilder(
    column: $table.version,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get thumbnailUrl => $composableBuilder(
    column: $table.thumbnailUrl,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get checksum => $composableBuilder(
    column: $table.checksum,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get status => $composableBuilder(
    column: $table.status,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get localDir => $composableBuilder(
    column: $table.localDir,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$LocalPacksTableOrderingComposer
    extends Composer<_$RestickDatabase, $LocalPacksTable> {
  $$LocalPacksTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get slug => $composableBuilder(
    column: $table.slug,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get name => $composableBuilder(
    column: $table.name,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get author => $composableBuilder(
    column: $table.author,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get tagsJson => $composableBuilder(
    column: $table.tagsJson,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get stickerCount => $composableBuilder(
    column: $table.stickerCount,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get version => $composableBuilder(
    column: $table.version,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get thumbnailUrl => $composableBuilder(
    column: $table.thumbnailUrl,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get checksum => $composableBuilder(
    column: $table.checksum,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get status => $composableBuilder(
    column: $table.status,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get localDir => $composableBuilder(
    column: $table.localDir,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$LocalPacksTableAnnotationComposer
    extends Composer<_$RestickDatabase, $LocalPacksTable> {
  $$LocalPacksTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get slug =>
      $composableBuilder(column: $table.slug, builder: (column) => column);

  GeneratedColumn<String> get name =>
      $composableBuilder(column: $table.name, builder: (column) => column);

  GeneratedColumn<String> get author =>
      $composableBuilder(column: $table.author, builder: (column) => column);

  GeneratedColumn<String> get tagsJson =>
      $composableBuilder(column: $table.tagsJson, builder: (column) => column);

  GeneratedColumn<int> get stickerCount => $composableBuilder(
    column: $table.stickerCount,
    builder: (column) => column,
  );

  GeneratedColumn<int> get version =>
      $composableBuilder(column: $table.version, builder: (column) => column);

  GeneratedColumn<String> get thumbnailUrl => $composableBuilder(
    column: $table.thumbnailUrl,
    builder: (column) => column,
  );

  GeneratedColumn<String> get checksum =>
      $composableBuilder(column: $table.checksum, builder: (column) => column);

  GeneratedColumn<String> get status =>
      $composableBuilder(column: $table.status, builder: (column) => column);

  GeneratedColumn<String> get localDir =>
      $composableBuilder(column: $table.localDir, builder: (column) => column);

  GeneratedColumn<DateTime> get updatedAt =>
      $composableBuilder(column: $table.updatedAt, builder: (column) => column);
}

class $$LocalPacksTableTableManager
    extends
        RootTableManager<
          _$RestickDatabase,
          $LocalPacksTable,
          LocalPack,
          $$LocalPacksTableFilterComposer,
          $$LocalPacksTableOrderingComposer,
          $$LocalPacksTableAnnotationComposer,
          $$LocalPacksTableCreateCompanionBuilder,
          $$LocalPacksTableUpdateCompanionBuilder,
          (
            LocalPack,
            BaseReferences<_$RestickDatabase, $LocalPacksTable, LocalPack>,
          ),
          LocalPack,
          PrefetchHooks Function()
        > {
  $$LocalPacksTableTableManager(_$RestickDatabase db, $LocalPacksTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer:
              () => $$LocalPacksTableFilterComposer($db: db, $table: table),
          createOrderingComposer:
              () => $$LocalPacksTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer:
              () => $$LocalPacksTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> slug = const Value.absent(),
                Value<String> name = const Value.absent(),
                Value<String> author = const Value.absent(),
                Value<String> tagsJson = const Value.absent(),
                Value<int> stickerCount = const Value.absent(),
                Value<int> version = const Value.absent(),
                Value<String> thumbnailUrl = const Value.absent(),
                Value<String> checksum = const Value.absent(),
                Value<String> status = const Value.absent(),
                Value<String> localDir = const Value.absent(),
                Value<DateTime> updatedAt = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => LocalPacksCompanion(
                id: id,
                slug: slug,
                name: name,
                author: author,
                tagsJson: tagsJson,
                stickerCount: stickerCount,
                version: version,
                thumbnailUrl: thumbnailUrl,
                checksum: checksum,
                status: status,
                localDir: localDir,
                updatedAt: updatedAt,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String slug,
                required String name,
                required String author,
                required String tagsJson,
                required int stickerCount,
                required int version,
                required String thumbnailUrl,
                required String checksum,
                required String status,
                required String localDir,
                required DateTime updatedAt,
                Value<int> rowid = const Value.absent(),
              }) => LocalPacksCompanion.insert(
                id: id,
                slug: slug,
                name: name,
                author: author,
                tagsJson: tagsJson,
                stickerCount: stickerCount,
                version: version,
                thumbnailUrl: thumbnailUrl,
                checksum: checksum,
                status: status,
                localDir: localDir,
                updatedAt: updatedAt,
                rowid: rowid,
              ),
          withReferenceMapper:
              (p0) =>
                  p0
                      .map(
                        (e) => (
                          e.readTable(table),
                          BaseReferences(db, table, e),
                        ),
                      )
                      .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$LocalPacksTableProcessedTableManager =
    ProcessedTableManager<
      _$RestickDatabase,
      $LocalPacksTable,
      LocalPack,
      $$LocalPacksTableFilterComposer,
      $$LocalPacksTableOrderingComposer,
      $$LocalPacksTableAnnotationComposer,
      $$LocalPacksTableCreateCompanionBuilder,
      $$LocalPacksTableUpdateCompanionBuilder,
      (
        LocalPack,
        BaseReferences<_$RestickDatabase, $LocalPacksTable, LocalPack>,
      ),
      LocalPack,
      PrefetchHooks Function()
    >;
typedef $$SyncEntriesTableCreateCompanionBuilder =
    SyncEntriesCompanion Function({
      required String key,
      required String value,
      Value<int> rowid,
    });
typedef $$SyncEntriesTableUpdateCompanionBuilder =
    SyncEntriesCompanion Function({
      Value<String> key,
      Value<String> value,
      Value<int> rowid,
    });

class $$SyncEntriesTableFilterComposer
    extends Composer<_$RestickDatabase, $SyncEntriesTable> {
  $$SyncEntriesTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get key => $composableBuilder(
    column: $table.key,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnFilters(column),
  );
}

class $$SyncEntriesTableOrderingComposer
    extends Composer<_$RestickDatabase, $SyncEntriesTable> {
  $$SyncEntriesTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get key => $composableBuilder(
    column: $table.key,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$SyncEntriesTableAnnotationComposer
    extends Composer<_$RestickDatabase, $SyncEntriesTable> {
  $$SyncEntriesTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get key =>
      $composableBuilder(column: $table.key, builder: (column) => column);

  GeneratedColumn<String> get value =>
      $composableBuilder(column: $table.value, builder: (column) => column);
}

class $$SyncEntriesTableTableManager
    extends
        RootTableManager<
          _$RestickDatabase,
          $SyncEntriesTable,
          SyncEntry,
          $$SyncEntriesTableFilterComposer,
          $$SyncEntriesTableOrderingComposer,
          $$SyncEntriesTableAnnotationComposer,
          $$SyncEntriesTableCreateCompanionBuilder,
          $$SyncEntriesTableUpdateCompanionBuilder,
          (
            SyncEntry,
            BaseReferences<_$RestickDatabase, $SyncEntriesTable, SyncEntry>,
          ),
          SyncEntry,
          PrefetchHooks Function()
        > {
  $$SyncEntriesTableTableManager(_$RestickDatabase db, $SyncEntriesTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer:
              () => $$SyncEntriesTableFilterComposer($db: db, $table: table),
          createOrderingComposer:
              () => $$SyncEntriesTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer:
              () =>
                  $$SyncEntriesTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> key = const Value.absent(),
                Value<String> value = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => SyncEntriesCompanion(key: key, value: value, rowid: rowid),
          createCompanionCallback:
              ({
                required String key,
                required String value,
                Value<int> rowid = const Value.absent(),
              }) => SyncEntriesCompanion.insert(
                key: key,
                value: value,
                rowid: rowid,
              ),
          withReferenceMapper:
              (p0) =>
                  p0
                      .map(
                        (e) => (
                          e.readTable(table),
                          BaseReferences(db, table, e),
                        ),
                      )
                      .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$SyncEntriesTableProcessedTableManager =
    ProcessedTableManager<
      _$RestickDatabase,
      $SyncEntriesTable,
      SyncEntry,
      $$SyncEntriesTableFilterComposer,
      $$SyncEntriesTableOrderingComposer,
      $$SyncEntriesTableAnnotationComposer,
      $$SyncEntriesTableCreateCompanionBuilder,
      $$SyncEntriesTableUpdateCompanionBuilder,
      (
        SyncEntry,
        BaseReferences<_$RestickDatabase, $SyncEntriesTable, SyncEntry>,
      ),
      SyncEntry,
      PrefetchHooks Function()
    >;

class $RestickDatabaseManager {
  final _$RestickDatabase _db;
  $RestickDatabaseManager(this._db);
  $$LocalPacksTableTableManager get localPacks =>
      $$LocalPacksTableTableManager(_db, _db.localPacks);
  $$SyncEntriesTableTableManager get syncEntries =>
      $$SyncEntriesTableTableManager(_db, _db.syncEntries);
}
