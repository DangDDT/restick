// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'app_database.dart';

// ignore_for_file: type=lint
class $CachedPacksTable extends CachedPacks
    with TableInfo<$CachedPacksTable, CachedPack> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CachedPacksTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _cacheKeyMeta = const VerificationMeta(
    'cacheKey',
  );
  @override
  late final GeneratedColumn<String> cacheKey = GeneratedColumn<String>(
    'cache_key',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _payloadJsonMeta = const VerificationMeta(
    'payloadJson',
  );
  @override
  late final GeneratedColumn<String> payloadJson = GeneratedColumn<String>(
    'payload_json',
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
  List<GeneratedColumn> get $columns => [cacheKey, payloadJson, updatedAt];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'cached_packs';
  @override
  VerificationContext validateIntegrity(
    Insertable<CachedPack> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('cache_key')) {
      context.handle(
        _cacheKeyMeta,
        cacheKey.isAcceptableOrUnknown(data['cache_key']!, _cacheKeyMeta),
      );
    } else if (isInserting) {
      context.missing(_cacheKeyMeta);
    }
    if (data.containsKey('payload_json')) {
      context.handle(
        _payloadJsonMeta,
        payloadJson.isAcceptableOrUnknown(
          data['payload_json']!,
          _payloadJsonMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_payloadJsonMeta);
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
  Set<GeneratedColumn> get $primaryKey => {cacheKey};
  @override
  CachedPack map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CachedPack(
      cacheKey:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}cache_key'],
          )!,
      payloadJson:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}payload_json'],
          )!,
      updatedAt:
          attachedDatabase.typeMapping.read(
            DriftSqlType.dateTime,
            data['${effectivePrefix}updated_at'],
          )!,
    );
  }

  @override
  $CachedPacksTable createAlias(String alias) {
    return $CachedPacksTable(attachedDatabase, alias);
  }
}

class CachedPack extends DataClass implements Insertable<CachedPack> {
  final String cacheKey;
  final String payloadJson;
  final DateTime updatedAt;
  const CachedPack({
    required this.cacheKey,
    required this.payloadJson,
    required this.updatedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['cache_key'] = Variable<String>(cacheKey);
    map['payload_json'] = Variable<String>(payloadJson);
    map['updated_at'] = Variable<DateTime>(updatedAt);
    return map;
  }

  CachedPacksCompanion toCompanion(bool nullToAbsent) {
    return CachedPacksCompanion(
      cacheKey: Value(cacheKey),
      payloadJson: Value(payloadJson),
      updatedAt: Value(updatedAt),
    );
  }

  factory CachedPack.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CachedPack(
      cacheKey: serializer.fromJson<String>(json['cacheKey']),
      payloadJson: serializer.fromJson<String>(json['payloadJson']),
      updatedAt: serializer.fromJson<DateTime>(json['updatedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'cacheKey': serializer.toJson<String>(cacheKey),
      'payloadJson': serializer.toJson<String>(payloadJson),
      'updatedAt': serializer.toJson<DateTime>(updatedAt),
    };
  }

  CachedPack copyWith({
    String? cacheKey,
    String? payloadJson,
    DateTime? updatedAt,
  }) => CachedPack(
    cacheKey: cacheKey ?? this.cacheKey,
    payloadJson: payloadJson ?? this.payloadJson,
    updatedAt: updatedAt ?? this.updatedAt,
  );
  CachedPack copyWithCompanion(CachedPacksCompanion data) {
    return CachedPack(
      cacheKey: data.cacheKey.present ? data.cacheKey.value : this.cacheKey,
      payloadJson:
          data.payloadJson.present ? data.payloadJson.value : this.payloadJson,
      updatedAt: data.updatedAt.present ? data.updatedAt.value : this.updatedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CachedPack(')
          ..write('cacheKey: $cacheKey, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('updatedAt: $updatedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(cacheKey, payloadJson, updatedAt);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CachedPack &&
          other.cacheKey == this.cacheKey &&
          other.payloadJson == this.payloadJson &&
          other.updatedAt == this.updatedAt);
}

class CachedPacksCompanion extends UpdateCompanion<CachedPack> {
  final Value<String> cacheKey;
  final Value<String> payloadJson;
  final Value<DateTime> updatedAt;
  final Value<int> rowid;
  const CachedPacksCompanion({
    this.cacheKey = const Value.absent(),
    this.payloadJson = const Value.absent(),
    this.updatedAt = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  CachedPacksCompanion.insert({
    required String cacheKey,
    required String payloadJson,
    required DateTime updatedAt,
    this.rowid = const Value.absent(),
  }) : cacheKey = Value(cacheKey),
       payloadJson = Value(payloadJson),
       updatedAt = Value(updatedAt);
  static Insertable<CachedPack> custom({
    Expression<String>? cacheKey,
    Expression<String>? payloadJson,
    Expression<DateTime>? updatedAt,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (cacheKey != null) 'cache_key': cacheKey,
      if (payloadJson != null) 'payload_json': payloadJson,
      if (updatedAt != null) 'updated_at': updatedAt,
      if (rowid != null) 'rowid': rowid,
    });
  }

  CachedPacksCompanion copyWith({
    Value<String>? cacheKey,
    Value<String>? payloadJson,
    Value<DateTime>? updatedAt,
    Value<int>? rowid,
  }) {
    return CachedPacksCompanion(
      cacheKey: cacheKey ?? this.cacheKey,
      payloadJson: payloadJson ?? this.payloadJson,
      updatedAt: updatedAt ?? this.updatedAt,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (cacheKey.present) {
      map['cache_key'] = Variable<String>(cacheKey.value);
    }
    if (payloadJson.present) {
      map['payload_json'] = Variable<String>(payloadJson.value);
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
    return (StringBuffer('CachedPacksCompanion(')
          ..write('cacheKey: $cacheKey, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('updatedAt: $updatedAt, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $CachedPackDetailsTable extends CachedPackDetails
    with TableInfo<$CachedPackDetailsTable, CachedPackDetail> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CachedPackDetailsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _packIdMeta = const VerificationMeta('packId');
  @override
  late final GeneratedColumn<String> packId = GeneratedColumn<String>(
    'pack_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _payloadJsonMeta = const VerificationMeta(
    'payloadJson',
  );
  @override
  late final GeneratedColumn<String> payloadJson = GeneratedColumn<String>(
    'payload_json',
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
  List<GeneratedColumn> get $columns => [packId, payloadJson, updatedAt];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'cached_pack_details';
  @override
  VerificationContext validateIntegrity(
    Insertable<CachedPackDetail> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('pack_id')) {
      context.handle(
        _packIdMeta,
        packId.isAcceptableOrUnknown(data['pack_id']!, _packIdMeta),
      );
    } else if (isInserting) {
      context.missing(_packIdMeta);
    }
    if (data.containsKey('payload_json')) {
      context.handle(
        _payloadJsonMeta,
        payloadJson.isAcceptableOrUnknown(
          data['payload_json']!,
          _payloadJsonMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_payloadJsonMeta);
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
  Set<GeneratedColumn> get $primaryKey => {packId};
  @override
  CachedPackDetail map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CachedPackDetail(
      packId:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}pack_id'],
          )!,
      payloadJson:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}payload_json'],
          )!,
      updatedAt:
          attachedDatabase.typeMapping.read(
            DriftSqlType.dateTime,
            data['${effectivePrefix}updated_at'],
          )!,
    );
  }

  @override
  $CachedPackDetailsTable createAlias(String alias) {
    return $CachedPackDetailsTable(attachedDatabase, alias);
  }
}

class CachedPackDetail extends DataClass
    implements Insertable<CachedPackDetail> {
  final String packId;
  final String payloadJson;
  final DateTime updatedAt;
  const CachedPackDetail({
    required this.packId,
    required this.payloadJson,
    required this.updatedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['pack_id'] = Variable<String>(packId);
    map['payload_json'] = Variable<String>(payloadJson);
    map['updated_at'] = Variable<DateTime>(updatedAt);
    return map;
  }

  CachedPackDetailsCompanion toCompanion(bool nullToAbsent) {
    return CachedPackDetailsCompanion(
      packId: Value(packId),
      payloadJson: Value(payloadJson),
      updatedAt: Value(updatedAt),
    );
  }

  factory CachedPackDetail.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CachedPackDetail(
      packId: serializer.fromJson<String>(json['packId']),
      payloadJson: serializer.fromJson<String>(json['payloadJson']),
      updatedAt: serializer.fromJson<DateTime>(json['updatedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'packId': serializer.toJson<String>(packId),
      'payloadJson': serializer.toJson<String>(payloadJson),
      'updatedAt': serializer.toJson<DateTime>(updatedAt),
    };
  }

  CachedPackDetail copyWith({
    String? packId,
    String? payloadJson,
    DateTime? updatedAt,
  }) => CachedPackDetail(
    packId: packId ?? this.packId,
    payloadJson: payloadJson ?? this.payloadJson,
    updatedAt: updatedAt ?? this.updatedAt,
  );
  CachedPackDetail copyWithCompanion(CachedPackDetailsCompanion data) {
    return CachedPackDetail(
      packId: data.packId.present ? data.packId.value : this.packId,
      payloadJson:
          data.payloadJson.present ? data.payloadJson.value : this.payloadJson,
      updatedAt: data.updatedAt.present ? data.updatedAt.value : this.updatedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CachedPackDetail(')
          ..write('packId: $packId, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('updatedAt: $updatedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(packId, payloadJson, updatedAt);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CachedPackDetail &&
          other.packId == this.packId &&
          other.payloadJson == this.payloadJson &&
          other.updatedAt == this.updatedAt);
}

class CachedPackDetailsCompanion extends UpdateCompanion<CachedPackDetail> {
  final Value<String> packId;
  final Value<String> payloadJson;
  final Value<DateTime> updatedAt;
  final Value<int> rowid;
  const CachedPackDetailsCompanion({
    this.packId = const Value.absent(),
    this.payloadJson = const Value.absent(),
    this.updatedAt = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  CachedPackDetailsCompanion.insert({
    required String packId,
    required String payloadJson,
    required DateTime updatedAt,
    this.rowid = const Value.absent(),
  }) : packId = Value(packId),
       payloadJson = Value(payloadJson),
       updatedAt = Value(updatedAt);
  static Insertable<CachedPackDetail> custom({
    Expression<String>? packId,
    Expression<String>? payloadJson,
    Expression<DateTime>? updatedAt,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (packId != null) 'pack_id': packId,
      if (payloadJson != null) 'payload_json': payloadJson,
      if (updatedAt != null) 'updated_at': updatedAt,
      if (rowid != null) 'rowid': rowid,
    });
  }

  CachedPackDetailsCompanion copyWith({
    Value<String>? packId,
    Value<String>? payloadJson,
    Value<DateTime>? updatedAt,
    Value<int>? rowid,
  }) {
    return CachedPackDetailsCompanion(
      packId: packId ?? this.packId,
      payloadJson: payloadJson ?? this.payloadJson,
      updatedAt: updatedAt ?? this.updatedAt,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (packId.present) {
      map['pack_id'] = Variable<String>(packId.value);
    }
    if (payloadJson.present) {
      map['payload_json'] = Variable<String>(payloadJson.value);
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
    return (StringBuffer('CachedPackDetailsCompanion(')
          ..write('packId: $packId, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('updatedAt: $updatedAt, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $CachedRejectionStatsTable extends CachedRejectionStats
    with TableInfo<$CachedRejectionStatsTable, CachedRejectionStat> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CachedRejectionStatsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _cacheKeyMeta = const VerificationMeta(
    'cacheKey',
  );
  @override
  late final GeneratedColumn<String> cacheKey = GeneratedColumn<String>(
    'cache_key',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _payloadJsonMeta = const VerificationMeta(
    'payloadJson',
  );
  @override
  late final GeneratedColumn<String> payloadJson = GeneratedColumn<String>(
    'payload_json',
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
  List<GeneratedColumn> get $columns => [cacheKey, payloadJson, updatedAt];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'cached_rejection_stats';
  @override
  VerificationContext validateIntegrity(
    Insertable<CachedRejectionStat> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('cache_key')) {
      context.handle(
        _cacheKeyMeta,
        cacheKey.isAcceptableOrUnknown(data['cache_key']!, _cacheKeyMeta),
      );
    } else if (isInserting) {
      context.missing(_cacheKeyMeta);
    }
    if (data.containsKey('payload_json')) {
      context.handle(
        _payloadJsonMeta,
        payloadJson.isAcceptableOrUnknown(
          data['payload_json']!,
          _payloadJsonMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_payloadJsonMeta);
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
  Set<GeneratedColumn> get $primaryKey => {cacheKey};
  @override
  CachedRejectionStat map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CachedRejectionStat(
      cacheKey:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}cache_key'],
          )!,
      payloadJson:
          attachedDatabase.typeMapping.read(
            DriftSqlType.string,
            data['${effectivePrefix}payload_json'],
          )!,
      updatedAt:
          attachedDatabase.typeMapping.read(
            DriftSqlType.dateTime,
            data['${effectivePrefix}updated_at'],
          )!,
    );
  }

  @override
  $CachedRejectionStatsTable createAlias(String alias) {
    return $CachedRejectionStatsTable(attachedDatabase, alias);
  }
}

class CachedRejectionStat extends DataClass
    implements Insertable<CachedRejectionStat> {
  final String cacheKey;
  final String payloadJson;
  final DateTime updatedAt;
  const CachedRejectionStat({
    required this.cacheKey,
    required this.payloadJson,
    required this.updatedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['cache_key'] = Variable<String>(cacheKey);
    map['payload_json'] = Variable<String>(payloadJson);
    map['updated_at'] = Variable<DateTime>(updatedAt);
    return map;
  }

  CachedRejectionStatsCompanion toCompanion(bool nullToAbsent) {
    return CachedRejectionStatsCompanion(
      cacheKey: Value(cacheKey),
      payloadJson: Value(payloadJson),
      updatedAt: Value(updatedAt),
    );
  }

  factory CachedRejectionStat.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CachedRejectionStat(
      cacheKey: serializer.fromJson<String>(json['cacheKey']),
      payloadJson: serializer.fromJson<String>(json['payloadJson']),
      updatedAt: serializer.fromJson<DateTime>(json['updatedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'cacheKey': serializer.toJson<String>(cacheKey),
      'payloadJson': serializer.toJson<String>(payloadJson),
      'updatedAt': serializer.toJson<DateTime>(updatedAt),
    };
  }

  CachedRejectionStat copyWith({
    String? cacheKey,
    String? payloadJson,
    DateTime? updatedAt,
  }) => CachedRejectionStat(
    cacheKey: cacheKey ?? this.cacheKey,
    payloadJson: payloadJson ?? this.payloadJson,
    updatedAt: updatedAt ?? this.updatedAt,
  );
  CachedRejectionStat copyWithCompanion(CachedRejectionStatsCompanion data) {
    return CachedRejectionStat(
      cacheKey: data.cacheKey.present ? data.cacheKey.value : this.cacheKey,
      payloadJson:
          data.payloadJson.present ? data.payloadJson.value : this.payloadJson,
      updatedAt: data.updatedAt.present ? data.updatedAt.value : this.updatedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CachedRejectionStat(')
          ..write('cacheKey: $cacheKey, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('updatedAt: $updatedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(cacheKey, payloadJson, updatedAt);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CachedRejectionStat &&
          other.cacheKey == this.cacheKey &&
          other.payloadJson == this.payloadJson &&
          other.updatedAt == this.updatedAt);
}

class CachedRejectionStatsCompanion
    extends UpdateCompanion<CachedRejectionStat> {
  final Value<String> cacheKey;
  final Value<String> payloadJson;
  final Value<DateTime> updatedAt;
  final Value<int> rowid;
  const CachedRejectionStatsCompanion({
    this.cacheKey = const Value.absent(),
    this.payloadJson = const Value.absent(),
    this.updatedAt = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  CachedRejectionStatsCompanion.insert({
    required String cacheKey,
    required String payloadJson,
    required DateTime updatedAt,
    this.rowid = const Value.absent(),
  }) : cacheKey = Value(cacheKey),
       payloadJson = Value(payloadJson),
       updatedAt = Value(updatedAt);
  static Insertable<CachedRejectionStat> custom({
    Expression<String>? cacheKey,
    Expression<String>? payloadJson,
    Expression<DateTime>? updatedAt,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (cacheKey != null) 'cache_key': cacheKey,
      if (payloadJson != null) 'payload_json': payloadJson,
      if (updatedAt != null) 'updated_at': updatedAt,
      if (rowid != null) 'rowid': rowid,
    });
  }

  CachedRejectionStatsCompanion copyWith({
    Value<String>? cacheKey,
    Value<String>? payloadJson,
    Value<DateTime>? updatedAt,
    Value<int>? rowid,
  }) {
    return CachedRejectionStatsCompanion(
      cacheKey: cacheKey ?? this.cacheKey,
      payloadJson: payloadJson ?? this.payloadJson,
      updatedAt: updatedAt ?? this.updatedAt,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (cacheKey.present) {
      map['cache_key'] = Variable<String>(cacheKey.value);
    }
    if (payloadJson.present) {
      map['payload_json'] = Variable<String>(payloadJson.value);
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
    return (StringBuffer('CachedRejectionStatsCompanion(')
          ..write('cacheKey: $cacheKey, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('updatedAt: $updatedAt, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

abstract class _$AppDatabase extends GeneratedDatabase {
  _$AppDatabase(QueryExecutor e) : super(e);
  $AppDatabaseManager get managers => $AppDatabaseManager(this);
  late final $CachedPacksTable cachedPacks = $CachedPacksTable(this);
  late final $CachedPackDetailsTable cachedPackDetails =
      $CachedPackDetailsTable(this);
  late final $CachedRejectionStatsTable cachedRejectionStats =
      $CachedRejectionStatsTable(this);
  @override
  Iterable<TableInfo<Table, Object?>> get allTables =>
      allSchemaEntities.whereType<TableInfo<Table, Object?>>();
  @override
  List<DatabaseSchemaEntity> get allSchemaEntities => [
    cachedPacks,
    cachedPackDetails,
    cachedRejectionStats,
  ];
  @override
  DriftDatabaseOptions get options =>
      const DriftDatabaseOptions(storeDateTimeAsText: true);
}

typedef $$CachedPacksTableCreateCompanionBuilder =
    CachedPacksCompanion Function({
      required String cacheKey,
      required String payloadJson,
      required DateTime updatedAt,
      Value<int> rowid,
    });
typedef $$CachedPacksTableUpdateCompanionBuilder =
    CachedPacksCompanion Function({
      Value<String> cacheKey,
      Value<String> payloadJson,
      Value<DateTime> updatedAt,
      Value<int> rowid,
    });

class $$CachedPacksTableFilterComposer
    extends Composer<_$AppDatabase, $CachedPacksTable> {
  $$CachedPacksTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get cacheKey => $composableBuilder(
    column: $table.cacheKey,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CachedPacksTableOrderingComposer
    extends Composer<_$AppDatabase, $CachedPacksTable> {
  $$CachedPacksTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get cacheKey => $composableBuilder(
    column: $table.cacheKey,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CachedPacksTableAnnotationComposer
    extends Composer<_$AppDatabase, $CachedPacksTable> {
  $$CachedPacksTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get cacheKey =>
      $composableBuilder(column: $table.cacheKey, builder: (column) => column);

  GeneratedColumn<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get updatedAt =>
      $composableBuilder(column: $table.updatedAt, builder: (column) => column);
}

class $$CachedPacksTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $CachedPacksTable,
          CachedPack,
          $$CachedPacksTableFilterComposer,
          $$CachedPacksTableOrderingComposer,
          $$CachedPacksTableAnnotationComposer,
          $$CachedPacksTableCreateCompanionBuilder,
          $$CachedPacksTableUpdateCompanionBuilder,
          (
            CachedPack,
            BaseReferences<_$AppDatabase, $CachedPacksTable, CachedPack>,
          ),
          CachedPack,
          PrefetchHooks Function()
        > {
  $$CachedPacksTableTableManager(_$AppDatabase db, $CachedPacksTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer:
              () => $$CachedPacksTableFilterComposer($db: db, $table: table),
          createOrderingComposer:
              () => $$CachedPacksTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer:
              () =>
                  $$CachedPacksTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> cacheKey = const Value.absent(),
                Value<String> payloadJson = const Value.absent(),
                Value<DateTime> updatedAt = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => CachedPacksCompanion(
                cacheKey: cacheKey,
                payloadJson: payloadJson,
                updatedAt: updatedAt,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String cacheKey,
                required String payloadJson,
                required DateTime updatedAt,
                Value<int> rowid = const Value.absent(),
              }) => CachedPacksCompanion.insert(
                cacheKey: cacheKey,
                payloadJson: payloadJson,
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

typedef $$CachedPacksTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $CachedPacksTable,
      CachedPack,
      $$CachedPacksTableFilterComposer,
      $$CachedPacksTableOrderingComposer,
      $$CachedPacksTableAnnotationComposer,
      $$CachedPacksTableCreateCompanionBuilder,
      $$CachedPacksTableUpdateCompanionBuilder,
      (
        CachedPack,
        BaseReferences<_$AppDatabase, $CachedPacksTable, CachedPack>,
      ),
      CachedPack,
      PrefetchHooks Function()
    >;
typedef $$CachedPackDetailsTableCreateCompanionBuilder =
    CachedPackDetailsCompanion Function({
      required String packId,
      required String payloadJson,
      required DateTime updatedAt,
      Value<int> rowid,
    });
typedef $$CachedPackDetailsTableUpdateCompanionBuilder =
    CachedPackDetailsCompanion Function({
      Value<String> packId,
      Value<String> payloadJson,
      Value<DateTime> updatedAt,
      Value<int> rowid,
    });

class $$CachedPackDetailsTableFilterComposer
    extends Composer<_$AppDatabase, $CachedPackDetailsTable> {
  $$CachedPackDetailsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get packId => $composableBuilder(
    column: $table.packId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CachedPackDetailsTableOrderingComposer
    extends Composer<_$AppDatabase, $CachedPackDetailsTable> {
  $$CachedPackDetailsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get packId => $composableBuilder(
    column: $table.packId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CachedPackDetailsTableAnnotationComposer
    extends Composer<_$AppDatabase, $CachedPackDetailsTable> {
  $$CachedPackDetailsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get packId =>
      $composableBuilder(column: $table.packId, builder: (column) => column);

  GeneratedColumn<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get updatedAt =>
      $composableBuilder(column: $table.updatedAt, builder: (column) => column);
}

class $$CachedPackDetailsTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $CachedPackDetailsTable,
          CachedPackDetail,
          $$CachedPackDetailsTableFilterComposer,
          $$CachedPackDetailsTableOrderingComposer,
          $$CachedPackDetailsTableAnnotationComposer,
          $$CachedPackDetailsTableCreateCompanionBuilder,
          $$CachedPackDetailsTableUpdateCompanionBuilder,
          (
            CachedPackDetail,
            BaseReferences<
              _$AppDatabase,
              $CachedPackDetailsTable,
              CachedPackDetail
            >,
          ),
          CachedPackDetail,
          PrefetchHooks Function()
        > {
  $$CachedPackDetailsTableTableManager(
    _$AppDatabase db,
    $CachedPackDetailsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer:
              () => $$CachedPackDetailsTableFilterComposer(
                $db: db,
                $table: table,
              ),
          createOrderingComposer:
              () => $$CachedPackDetailsTableOrderingComposer(
                $db: db,
                $table: table,
              ),
          createComputedFieldComposer:
              () => $$CachedPackDetailsTableAnnotationComposer(
                $db: db,
                $table: table,
              ),
          updateCompanionCallback:
              ({
                Value<String> packId = const Value.absent(),
                Value<String> payloadJson = const Value.absent(),
                Value<DateTime> updatedAt = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => CachedPackDetailsCompanion(
                packId: packId,
                payloadJson: payloadJson,
                updatedAt: updatedAt,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String packId,
                required String payloadJson,
                required DateTime updatedAt,
                Value<int> rowid = const Value.absent(),
              }) => CachedPackDetailsCompanion.insert(
                packId: packId,
                payloadJson: payloadJson,
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

typedef $$CachedPackDetailsTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $CachedPackDetailsTable,
      CachedPackDetail,
      $$CachedPackDetailsTableFilterComposer,
      $$CachedPackDetailsTableOrderingComposer,
      $$CachedPackDetailsTableAnnotationComposer,
      $$CachedPackDetailsTableCreateCompanionBuilder,
      $$CachedPackDetailsTableUpdateCompanionBuilder,
      (
        CachedPackDetail,
        BaseReferences<
          _$AppDatabase,
          $CachedPackDetailsTable,
          CachedPackDetail
        >,
      ),
      CachedPackDetail,
      PrefetchHooks Function()
    >;
typedef $$CachedRejectionStatsTableCreateCompanionBuilder =
    CachedRejectionStatsCompanion Function({
      required String cacheKey,
      required String payloadJson,
      required DateTime updatedAt,
      Value<int> rowid,
    });
typedef $$CachedRejectionStatsTableUpdateCompanionBuilder =
    CachedRejectionStatsCompanion Function({
      Value<String> cacheKey,
      Value<String> payloadJson,
      Value<DateTime> updatedAt,
      Value<int> rowid,
    });

class $$CachedRejectionStatsTableFilterComposer
    extends Composer<_$AppDatabase, $CachedRejectionStatsTable> {
  $$CachedRejectionStatsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get cacheKey => $composableBuilder(
    column: $table.cacheKey,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CachedRejectionStatsTableOrderingComposer
    extends Composer<_$AppDatabase, $CachedRejectionStatsTable> {
  $$CachedRejectionStatsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get cacheKey => $composableBuilder(
    column: $table.cacheKey,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get updatedAt => $composableBuilder(
    column: $table.updatedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CachedRejectionStatsTableAnnotationComposer
    extends Composer<_$AppDatabase, $CachedRejectionStatsTable> {
  $$CachedRejectionStatsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get cacheKey =>
      $composableBuilder(column: $table.cacheKey, builder: (column) => column);

  GeneratedColumn<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get updatedAt =>
      $composableBuilder(column: $table.updatedAt, builder: (column) => column);
}

class $$CachedRejectionStatsTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $CachedRejectionStatsTable,
          CachedRejectionStat,
          $$CachedRejectionStatsTableFilterComposer,
          $$CachedRejectionStatsTableOrderingComposer,
          $$CachedRejectionStatsTableAnnotationComposer,
          $$CachedRejectionStatsTableCreateCompanionBuilder,
          $$CachedRejectionStatsTableUpdateCompanionBuilder,
          (
            CachedRejectionStat,
            BaseReferences<
              _$AppDatabase,
              $CachedRejectionStatsTable,
              CachedRejectionStat
            >,
          ),
          CachedRejectionStat,
          PrefetchHooks Function()
        > {
  $$CachedRejectionStatsTableTableManager(
    _$AppDatabase db,
    $CachedRejectionStatsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer:
              () => $$CachedRejectionStatsTableFilterComposer(
                $db: db,
                $table: table,
              ),
          createOrderingComposer:
              () => $$CachedRejectionStatsTableOrderingComposer(
                $db: db,
                $table: table,
              ),
          createComputedFieldComposer:
              () => $$CachedRejectionStatsTableAnnotationComposer(
                $db: db,
                $table: table,
              ),
          updateCompanionCallback:
              ({
                Value<String> cacheKey = const Value.absent(),
                Value<String> payloadJson = const Value.absent(),
                Value<DateTime> updatedAt = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => CachedRejectionStatsCompanion(
                cacheKey: cacheKey,
                payloadJson: payloadJson,
                updatedAt: updatedAt,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String cacheKey,
                required String payloadJson,
                required DateTime updatedAt,
                Value<int> rowid = const Value.absent(),
              }) => CachedRejectionStatsCompanion.insert(
                cacheKey: cacheKey,
                payloadJson: payloadJson,
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

typedef $$CachedRejectionStatsTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $CachedRejectionStatsTable,
      CachedRejectionStat,
      $$CachedRejectionStatsTableFilterComposer,
      $$CachedRejectionStatsTableOrderingComposer,
      $$CachedRejectionStatsTableAnnotationComposer,
      $$CachedRejectionStatsTableCreateCompanionBuilder,
      $$CachedRejectionStatsTableUpdateCompanionBuilder,
      (
        CachedRejectionStat,
        BaseReferences<
          _$AppDatabase,
          $CachedRejectionStatsTable,
          CachedRejectionStat
        >,
      ),
      CachedRejectionStat,
      PrefetchHooks Function()
    >;

class $AppDatabaseManager {
  final _$AppDatabase _db;
  $AppDatabaseManager(this._db);
  $$CachedPacksTableTableManager get cachedPacks =>
      $$CachedPacksTableTableManager(_db, _db.cachedPacks);
  $$CachedPackDetailsTableTableManager get cachedPackDetails =>
      $$CachedPackDetailsTableTableManager(_db, _db.cachedPackDetails);
  $$CachedRejectionStatsTableTableManager get cachedRejectionStats =>
      $$CachedRejectionStatsTableTableManager(_db, _db.cachedRejectionStats);
}
