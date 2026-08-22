import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_vi.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('vi'),
  ];

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'Restick'**
  String get appTitle;

  /// No description provided for @navContribHeader.
  ///
  /// In en, this message translates to:
  /// **'Contributor'**
  String get navContribHeader;

  /// No description provided for @navMyPacks.
  ///
  /// In en, this message translates to:
  /// **'My packs'**
  String get navMyPacks;

  /// No description provided for @navUploadPack.
  ///
  /// In en, this message translates to:
  /// **'Upload'**
  String get navUploadPack;

  /// No description provided for @navAdminHeader.
  ///
  /// In en, this message translates to:
  /// **'Admin'**
  String get navAdminHeader;

  /// No description provided for @navQueue.
  ///
  /// In en, this message translates to:
  /// **'Review queue'**
  String get navQueue;

  /// No description provided for @navCatalog.
  ///
  /// In en, this message translates to:
  /// **'Published'**
  String get navCatalog;

  /// No description provided for @navAnalytics.
  ///
  /// In en, this message translates to:
  /// **'Rejection stats'**
  String get navAnalytics;

  /// No description provided for @navApiDocs.
  ///
  /// In en, this message translates to:
  /// **'API docs'**
  String get navApiDocs;

  /// No description provided for @titleApiDocs.
  ///
  /// In en, this message translates to:
  /// **'API documentation'**
  String get titleApiDocs;

  /// No description provided for @descApiDocs.
  ///
  /// In en, this message translates to:
  /// **'Swagger for every HTTP REST endpoint api_svc currently serves. Use Authorize to attach a Bearer JWT or X-Api-Key, then try calls.'**
  String get descApiDocs;

  /// No description provided for @titleMyPacks.
  ///
  /// In en, this message translates to:
  /// **'My sticker packs'**
  String get titleMyPacks;

  /// No description provided for @descMyPacks.
  ///
  /// In en, this message translates to:
  /// **'Track uploads, processing, and publish status.'**
  String get descMyPacks;

  /// No description provided for @btnUpload.
  ///
  /// In en, this message translates to:
  /// **'New pack'**
  String get btnUpload;

  /// No description provided for @titleNewPack.
  ///
  /// In en, this message translates to:
  /// **'Upload a sticker pack'**
  String get titleNewPack;

  /// No description provided for @descNewPack.
  ///
  /// In en, this message translates to:
  /// **'Drop in one sheet to auto-cut, or upload separate PNG/WebP files.'**
  String get descNewPack;

  /// No description provided for @noticeHeader.
  ///
  /// In en, this message translates to:
  /// **'Heads up'**
  String get noticeHeader;

  /// No description provided for @noticeBody.
  ///
  /// In en, this message translates to:
  /// **'Restick doesn’t generate images from prompts. Upload stickers you’ve already made.'**
  String get noticeBody;

  /// No description provided for @labelTitle.
  ///
  /// In en, this message translates to:
  /// **'Pack name'**
  String get labelTitle;

  /// No description provided for @labelSlug.
  ///
  /// In en, this message translates to:
  /// **'Slug'**
  String get labelSlug;

  /// No description provided for @labelAuthor.
  ///
  /// In en, this message translates to:
  /// **'Author'**
  String get labelAuthor;

  /// No description provided for @labelTags.
  ///
  /// In en, this message translates to:
  /// **'Tags (comma-separated)'**
  String get labelTags;

  /// No description provided for @labelSource.
  ///
  /// In en, this message translates to:
  /// **'Source'**
  String get labelSource;

  /// No description provided for @sourceSheet.
  ///
  /// In en, this message translates to:
  /// **'Single sheet (auto-cut)'**
  String get sourceSheet;

  /// No description provided for @sourceFiles.
  ///
  /// In en, this message translates to:
  /// **'Separate files (.png / .webp)'**
  String get sourceFiles;

  /// No description provided for @labelUpload.
  ///
  /// In en, this message translates to:
  /// **'Artwork'**
  String get labelUpload;

  /// No description provided for @dropzoneText.
  ///
  /// In en, this message translates to:
  /// **'Click or drop images here'**
  String get dropzoneText;

  /// No description provided for @dropzoneHint.
  ///
  /// In en, this message translates to:
  /// **'PNG, WEBP, or JPG up to 15MB'**
  String get dropzoneHint;

  /// No description provided for @btnCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get btnCancel;

  /// No description provided for @btnSubmit.
  ///
  /// In en, this message translates to:
  /// **'Submit for processing'**
  String get btnSubmit;

  /// No description provided for @sectionStickers.
  ///
  /// In en, this message translates to:
  /// **'Normalized stickers (WebP 512×512)'**
  String get sectionStickers;

  /// No description provided for @hintClickSend.
  ///
  /// In en, this message translates to:
  /// **'Click a sticker to preview it in chat'**
  String get hintClickSend;

  /// No description provided for @bannerConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Publish now'**
  String get bannerConfirmTitle;

  /// No description provided for @bannerConfirmBody.
  ///
  /// In en, this message translates to:
  /// **'This pack was uploaded by an admin, so you can publish it without the review queue.'**
  String get bannerConfirmBody;

  /// No description provided for @btnConfirmPublish.
  ///
  /// In en, this message translates to:
  /// **'Publish now'**
  String get btnConfirmPublish;

  /// No description provided for @titleQueue.
  ///
  /// In en, this message translates to:
  /// **'Review queue'**
  String get titleQueue;

  /// No description provided for @descQueue.
  ///
  /// In en, this message translates to:
  /// **'Check copyright and safety before packs go live.'**
  String get descQueue;

  /// No description provided for @btnApprove.
  ///
  /// In en, this message translates to:
  /// **'Approve'**
  String get btnApprove;

  /// No description provided for @btnReject.
  ///
  /// In en, this message translates to:
  /// **'Reject'**
  String get btnReject;

  /// No description provided for @titleCatalog.
  ///
  /// In en, this message translates to:
  /// **'Published packs'**
  String get titleCatalog;

  /// No description provided for @descCatalog.
  ///
  /// In en, this message translates to:
  /// **'Packs currently served to Flutter SDK clients.'**
  String get descCatalog;

  /// No description provided for @btnRevoke.
  ///
  /// In en, this message translates to:
  /// **'Revoke'**
  String get btnRevoke;

  /// No description provided for @titleAnalytics.
  ///
  /// In en, this message translates to:
  /// **'Rejection stats'**
  String get titleAnalytics;

  /// No description provided for @descAnalytics.
  ///
  /// In en, this message translates to:
  /// **'How often packs are rejected by reason.'**
  String get descAnalytics;

  /// No description provided for @chartTitle.
  ///
  /// In en, this message translates to:
  /// **'Rejection rate by reason'**
  String get chartTitle;

  /// No description provided for @btnRefresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get btnRefresh;

  /// No description provided for @deviceTitle.
  ///
  /// In en, this message translates to:
  /// **'Device preview'**
  String get deviceTitle;

  /// No description provided for @deviceSub.
  ///
  /// In en, this message translates to:
  /// **'How stickers look in a chat app'**
  String get deviceSub;

  /// No description provided for @msgIntro.
  ///
  /// In en, this message translates to:
  /// **'Hi! This frame simulates an app built with the Restick Flutter SDK.'**
  String get msgIntro;

  /// No description provided for @msgPrompt.
  ///
  /// In en, this message translates to:
  /// **'Click a sticker on the left to send it into the chat.'**
  String get msgPrompt;

  /// No description provided for @signIn.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get signIn;

  /// No description provided for @signUp.
  ///
  /// In en, this message translates to:
  /// **'Create account'**
  String get signUp;

  /// No description provided for @signOut.
  ///
  /// In en, this message translates to:
  /// **'Sign out'**
  String get signOut;

  /// No description provided for @email.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get email;

  /// No description provided for @password.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get password;

  /// No description provided for @displayName.
  ///
  /// In en, this message translates to:
  /// **'Display name'**
  String get displayName;

  /// No description provided for @loginSubtitle.
  ///
  /// In en, this message translates to:
  /// **'No keyword-to-image generation — upload art made elsewhere.'**
  String get loginSubtitle;

  /// No description provided for @themeDark.
  ///
  /// In en, this message translates to:
  /// **'Dark mode'**
  String get themeDark;

  /// No description provided for @themeLight.
  ///
  /// In en, this message translates to:
  /// **'Light mode'**
  String get themeLight;

  /// No description provided for @language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// No description provided for @modalRejectTitle.
  ///
  /// In en, this message translates to:
  /// **'Reject this pack'**
  String get modalRejectTitle;

  /// No description provided for @modalRejectBody.
  ///
  /// In en, this message translates to:
  /// **'Tell the contributor why. They’ll see this message.'**
  String get modalRejectBody;

  /// No description provided for @modalRevokeTitle.
  ///
  /// In en, this message translates to:
  /// **'Revoke published pack'**
  String get modalRevokeTitle;

  /// No description provided for @modalRevokeBody.
  ///
  /// In en, this message translates to:
  /// **'Devices will be told to remove this pack. Please include a reason.'**
  String get modalRevokeBody;

  /// No description provided for @labelQuickPresets.
  ///
  /// In en, this message translates to:
  /// **'Quick picks:'**
  String get labelQuickPresets;

  /// No description provided for @presetCopyright.
  ///
  /// In en, this message translates to:
  /// **'Copyright / IP'**
  String get presetCopyright;

  /// No description provided for @presetQuality.
  ///
  /// In en, this message translates to:
  /// **'Low quality / pixelated'**
  String get presetQuality;

  /// No description provided for @presetDuplicate.
  ///
  /// In en, this message translates to:
  /// **'Duplicate art'**
  String get presetDuplicate;

  /// No description provided for @presetMisleading.
  ///
  /// In en, this message translates to:
  /// **'Misleading metadata'**
  String get presetMisleading;

  /// No description provided for @presetExplicit.
  ///
  /// In en, this message translates to:
  /// **'Unsafe content'**
  String get presetExplicit;

  /// No description provided for @presetDmca.
  ///
  /// In en, this message translates to:
  /// **'DMCA take-down'**
  String get presetDmca;

  /// No description provided for @presetMalicious.
  ///
  /// In en, this message translates to:
  /// **'Malicious / abuse'**
  String get presetMalicious;

  /// No description provided for @presetAuthorRequest.
  ///
  /// In en, this message translates to:
  /// **'Author withdrawal'**
  String get presetAuthorRequest;

  /// No description provided for @presetTos.
  ///
  /// In en, this message translates to:
  /// **'Terms violation'**
  String get presetTos;

  /// No description provided for @labelReason.
  ///
  /// In en, this message translates to:
  /// **'Reason'**
  String get labelReason;

  /// No description provided for @btnSubmitReason.
  ///
  /// In en, this message translates to:
  /// **'Submit'**
  String get btnSubmitReason;

  /// No description provided for @publishTitle.
  ///
  /// In en, this message translates to:
  /// **'Publish this pack?'**
  String get publishTitle;

  /// No description provided for @publishBody.
  ///
  /// In en, this message translates to:
  /// **'Once published, stickers show up in the Flutter SDK.'**
  String get publishBody;

  /// No description provided for @btnPublish.
  ///
  /// In en, this message translates to:
  /// **'Publish'**
  String get btnPublish;

  /// No description provided for @emptyPacks.
  ///
  /// In en, this message translates to:
  /// **'No packs yet. Upload one to get started.'**
  String get emptyPacks;

  /// No description provided for @emptyQueue.
  ///
  /// In en, this message translates to:
  /// **'Nothing waiting for review.'**
  String get emptyQueue;

  /// No description provided for @emptyPublished.
  ///
  /// In en, this message translates to:
  /// **'No published packs yet.'**
  String get emptyPublished;

  /// No description provided for @emptyAnalytics.
  ///
  /// In en, this message translates to:
  /// **'No rejection data yet.'**
  String get emptyAnalytics;

  /// No description provided for @statusDraft.
  ///
  /// In en, this message translates to:
  /// **'Draft'**
  String get statusDraft;

  /// No description provided for @statusProcessing.
  ///
  /// In en, this message translates to:
  /// **'Processing'**
  String get statusProcessing;

  /// No description provided for @statusConfirm.
  ///
  /// In en, this message translates to:
  /// **'Awaiting confirm'**
  String get statusConfirm;

  /// No description provided for @statusPending.
  ///
  /// In en, this message translates to:
  /// **'Pending review'**
  String get statusPending;

  /// No description provided for @statusPublished.
  ///
  /// In en, this message translates to:
  /// **'Published'**
  String get statusPublished;

  /// No description provided for @statusRejected.
  ///
  /// In en, this message translates to:
  /// **'Rejected'**
  String get statusRejected;

  /// No description provided for @statusRevoked.
  ///
  /// In en, this message translates to:
  /// **'Revoked'**
  String get statusRevoked;

  /// No description provided for @filesSelected.
  ///
  /// In en, this message translates to:
  /// **'{count} file(s) selected'**
  String filesSelected(int count);

  /// No description provided for @chooseImages.
  ///
  /// In en, this message translates to:
  /// **'Choose images'**
  String get chooseImages;

  /// No description provided for @uploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading…'**
  String get uploading;

  /// No description provided for @signingIn.
  ///
  /// In en, this message translates to:
  /// **'Signing in…'**
  String get signingIn;

  /// No description provided for @creatingAccount.
  ///
  /// In en, this message translates to:
  /// **'Creating account…'**
  String get creatingAccount;

  /// No description provided for @roleAdmin.
  ///
  /// In en, this message translates to:
  /// **'Admin'**
  String get roleAdmin;

  /// No description provided for @roleContributor.
  ///
  /// In en, this message translates to:
  /// **'Contributor'**
  String get roleContributor;

  /// No description provided for @rejectReasonLabel.
  ///
  /// In en, this message translates to:
  /// **'Rejection reason: {reason}'**
  String rejectReasonLabel(String reason);

  /// No description provided for @packSlugMeta.
  ///
  /// In en, this message translates to:
  /// **'{slug} · {count} stickers'**
  String packSlugMeta(String slug, int count);

  /// No description provided for @menuDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get menuDelete;

  /// No description provided for @confirmDeleteRejectedTitle.
  ///
  /// In en, this message translates to:
  /// **'Delete permanently?'**
  String get confirmDeleteRejectedTitle;

  /// No description provided for @confirmDeleteRejectedBody.
  ///
  /// In en, this message translates to:
  /// **'This rejected pack will be removed for good. You can’t undo this.'**
  String get confirmDeleteRejectedBody;

  /// No description provided for @confirmRevokePublishedTitle.
  ///
  /// In en, this message translates to:
  /// **'Withdraw this published pack?'**
  String get confirmRevokePublishedTitle;

  /// No description provided for @confirmRevokePublishedBody.
  ///
  /// In en, this message translates to:
  /// **'It will leave the public catalog. Devices get a tombstone so they can clear local copies; audit history stays.'**
  String get confirmRevokePublishedBody;

  /// No description provided for @btnConfirmDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete forever'**
  String get btnConfirmDelete;

  /// No description provided for @btnConfirmRevoke.
  ///
  /// In en, this message translates to:
  /// **'Withdraw'**
  String get btnConfirmRevoke;

  /// No description provided for @navApiKeys.
  ///
  /// In en, this message translates to:
  /// **'API Keys'**
  String get navApiKeys;

  /// No description provided for @titleApiKeys.
  ///
  /// In en, this message translates to:
  /// **'API Keys'**
  String get titleApiKeys;

  /// No description provided for @descApiKeys.
  ///
  /// In en, this message translates to:
  /// **'Manage personal and machine API keys for client and SDK access.'**
  String get descApiKeys;

  /// No description provided for @btnCreateApiKey.
  ///
  /// In en, this message translates to:
  /// **'Create key'**
  String get btnCreateApiKey;

  /// No description provided for @labelKeyName.
  ///
  /// In en, this message translates to:
  /// **'Key name / label'**
  String get labelKeyName;

  /// No description provided for @labelExpiresInDays.
  ///
  /// In en, this message translates to:
  /// **'Expiration in days (optional)'**
  String get labelExpiresInDays;

  /// No description provided for @labelExpiresNever.
  ///
  /// In en, this message translates to:
  /// **'Never expires'**
  String get labelExpiresNever;

  /// No description provided for @labelKeyPrefix.
  ///
  /// In en, this message translates to:
  /// **'Token prefix'**
  String get labelKeyPrefix;

  /// No description provided for @labelCreatedAt.
  ///
  /// In en, this message translates to:
  /// **'Created at'**
  String get labelCreatedAt;

  /// No description provided for @labelExpiresAt.
  ///
  /// In en, this message translates to:
  /// **'Expires at'**
  String get labelExpiresAt;

  /// No description provided for @labelLastUsedAt.
  ///
  /// In en, this message translates to:
  /// **'Last used'**
  String get labelLastUsedAt;

  /// No description provided for @labelNeverUsed.
  ///
  /// In en, this message translates to:
  /// **'Never'**
  String get labelNeverUsed;

  /// No description provided for @btnRotateKey.
  ///
  /// In en, this message translates to:
  /// **'Rotate key'**
  String get btnRotateKey;

  /// No description provided for @btnDeleteKey.
  ///
  /// In en, this message translates to:
  /// **'Revoke & Delete'**
  String get btnDeleteKey;

  /// No description provided for @dialogCreatedKeyTitle.
  ///
  /// In en, this message translates to:
  /// **'Save your API Key'**
  String get dialogCreatedKeyTitle;

  /// No description provided for @dialogCreatedKeyWarning.
  ///
  /// In en, this message translates to:
  /// **'Copy this key now. For your security, you will not be able to see it again!'**
  String get dialogCreatedKeyWarning;

  /// No description provided for @btnCopyKey.
  ///
  /// In en, this message translates to:
  /// **'Copy to clipboard'**
  String get btnCopyKey;

  /// No description provided for @copiedToClipboard.
  ///
  /// In en, this message translates to:
  /// **'Copied to clipboard!'**
  String get copiedToClipboard;

  /// No description provided for @confirmRotateKeyTitle.
  ///
  /// In en, this message translates to:
  /// **'Rotate this API Key?'**
  String get confirmRotateKeyTitle;

  /// No description provided for @confirmRotateKeyBody.
  ///
  /// In en, this message translates to:
  /// **'The existing key will immediately stop working and a new key will be generated.'**
  String get confirmRotateKeyBody;

  /// No description provided for @confirmDeleteKeyTitle.
  ///
  /// In en, this message translates to:
  /// **'Delete and revoke this API Key?'**
  String get confirmDeleteKeyTitle;

  /// No description provided for @confirmDeleteKeyBody.
  ///
  /// In en, this message translates to:
  /// **'Any client or SDK using this API key will be permanently unauthorized.'**
  String get confirmDeleteKeyBody;

  /// No description provided for @emptyApiKeys.
  ///
  /// In en, this message translates to:
  /// **'No API keys found. Click \'Create key\' to issue one.'**
  String get emptyApiKeys;

  /// No description provided for @tabPersonalKeys.
  ///
  /// In en, this message translates to:
  /// **'Personal keys'**
  String get tabPersonalKeys;

  /// No description provided for @tabSystemKeys.
  ///
  /// In en, this message translates to:
  /// **'All system keys (Admin)'**
  String get tabSystemKeys;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'vi'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'vi':
      return AppLocalizationsVi();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
