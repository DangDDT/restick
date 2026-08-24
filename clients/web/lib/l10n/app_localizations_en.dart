// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Restick';

  @override
  String get navContribHeader => 'Contributor';

  @override
  String get navMyPacks => 'My packs';

  @override
  String get navUploadPack => 'Upload';

  @override
  String get navAdminHeader => 'Admin';

  @override
  String get navQueue => 'Review queue';

  @override
  String get navCatalog => 'Published';

  @override
  String get navAnalytics => 'Rejection stats';

  @override
  String get navApiDocs => 'API docs';

  @override
  String get titleApiDocs => 'API documentation';

  @override
  String get descApiDocs =>
      'Swagger for every HTTP REST endpoint api_svc currently serves. Use Authorize to attach a Bearer JWT or X-Api-Key, then try calls.';

  @override
  String get titleMyPacks => 'My sticker packs';

  @override
  String get descMyPacks => 'Track uploads, processing, and publish status.';

  @override
  String get btnUpload => 'New pack';

  @override
  String get titleNewPack => 'Upload a sticker pack';

  @override
  String get descNewPack =>
      'Drop in one sheet to auto-cut, or upload separate PNG/WebP files.';

  @override
  String get noticeHeader => 'Heads up';

  @override
  String get noticeBody =>
      'Restick doesn’t generate images from prompts. Upload stickers you’ve already made.';

  @override
  String get labelTitle => 'Pack name';

  @override
  String get labelSlug => 'Slug';

  @override
  String get labelAuthor => 'Author';

  @override
  String get labelTags => 'Tags (comma-separated)';

  @override
  String get labelSource => 'Source';

  @override
  String get sourceSheet => 'Single sheet (auto-cut)';

  @override
  String get sourceFiles => 'Separate files (.png / .webp)';

  @override
  String get labelUpload => 'Artwork';

  @override
  String get dropzoneText => 'Click or drop images here';

  @override
  String get dropzoneHint => 'PNG, WEBP, or JPG up to 15MB';

  @override
  String get btnCancel => 'Cancel';

  @override
  String get btnSubmit => 'Submit for processing';

  @override
  String get sectionStickers => 'Normalized stickers (WebP 512×512)';

  @override
  String get hintClickSend => 'Click a sticker to preview it in chat';

  @override
  String get bannerConfirmTitle => 'Publish now';

  @override
  String get bannerConfirmBody =>
      'This pack was uploaded by an admin, so you can publish it without the review queue.';

  @override
  String get btnConfirmPublish => 'Publish now';

  @override
  String get titleQueue => 'Review queue';

  @override
  String get descQueue => 'Check copyright and safety before packs go live.';

  @override
  String get btnApprove => 'Approve';

  @override
  String get btnReject => 'Reject';

  @override
  String get titleCatalog => 'Published packs';

  @override
  String get descCatalog => 'Packs currently served to Flutter SDK clients.';

  @override
  String get btnRevoke => 'Revoke';

  @override
  String get titleAnalytics => 'Rejection stats';

  @override
  String get descAnalytics => 'How often packs are rejected by reason.';

  @override
  String get chartTitle => 'Rejection rate by reason';

  @override
  String get btnRefresh => 'Refresh';

  @override
  String get deviceTitle => 'Device preview';

  @override
  String get deviceSub => 'How stickers look in a chat app';

  @override
  String get msgIntro =>
      'Hi! This frame simulates an app built with the Restick Flutter SDK.';

  @override
  String get msgPrompt =>
      'Click a sticker on the left to send it into the chat.';

  @override
  String get signIn => 'Sign in';

  @override
  String get signUp => 'Create account';

  @override
  String get signOut => 'Sign out';

  @override
  String get email => 'Email';

  @override
  String get password => 'Password';

  @override
  String get displayName => 'Display name';

  @override
  String get loginSubtitle =>
      'No keyword-to-image generation — upload art made elsewhere.';

  @override
  String get themeDark => 'Dark mode';

  @override
  String get themeLight => 'Light mode';

  @override
  String get language => 'Language';

  @override
  String get modalRejectTitle => 'Reject this pack';

  @override
  String get modalRejectBody =>
      'Tell the contributor why. They’ll see this message.';

  @override
  String get modalRevokeTitle => 'Revoke published pack';

  @override
  String get modalRevokeBody =>
      'Devices will be told to remove this pack. Please include a reason.';

  @override
  String get labelQuickPresets => 'Quick picks:';

  @override
  String get presetCopyright => 'Copyright / IP';

  @override
  String get presetQuality => 'Low quality / pixelated';

  @override
  String get presetDuplicate => 'Duplicate art';

  @override
  String get presetMisleading => 'Misleading metadata';

  @override
  String get presetExplicit => 'Unsafe content';

  @override
  String get presetDmca => 'DMCA take-down';

  @override
  String get presetMalicious => 'Malicious / abuse';

  @override
  String get presetAuthorRequest => 'Author withdrawal';

  @override
  String get presetTos => 'Terms violation';

  @override
  String get labelReason => 'Reason';

  @override
  String get btnSubmitReason => 'Submit';

  @override
  String get publishTitle => 'Publish this pack?';

  @override
  String get publishBody =>
      'Once published, stickers show up in the Flutter SDK.';

  @override
  String get btnPublish => 'Publish';

  @override
  String get emptyPacks => 'No packs yet. Upload one to get started.';

  @override
  String get emptyQueue => 'Nothing waiting for review.';

  @override
  String get emptyPublished => 'No published packs yet.';

  @override
  String get emptyAnalytics => 'No rejection data yet.';

  @override
  String get statusDraft => 'Draft';

  @override
  String get statusProcessing => 'Processing';

  @override
  String get statusConfirm => 'Awaiting confirm';

  @override
  String get statusPending => 'Pending review';

  @override
  String get statusPublished => 'Published';

  @override
  String get statusRejected => 'Rejected';

  @override
  String get statusRevoked => 'Revoked';

  @override
  String filesSelected(int count) {
    return '$count file(s) selected';
  }

  @override
  String get chooseImages => 'Choose images';

  @override
  String get uploading => 'Uploading…';

  @override
  String get signingIn => 'Signing in…';

  @override
  String get creatingAccount => 'Creating account…';

  @override
  String get roleAdmin => 'Admin';

  @override
  String get roleContributor => 'Contributor';

  @override
  String rejectReasonLabel(String reason) {
    return 'Rejection reason: $reason';
  }

  @override
  String packSlugMeta(String slug, int count) {
    return '$slug · $count stickers';
  }

  @override
  String get menuDelete => 'Delete';

  @override
  String get confirmDeleteRejectedTitle => 'Delete permanently?';

  @override
  String get confirmDeleteRejectedBody =>
      'This rejected pack will be removed for good. You can’t undo this.';

  @override
  String get confirmRevokePublishedTitle => 'Withdraw this published pack?';

  @override
  String get confirmRevokePublishedBody =>
      'It will leave the public catalog. Devices get a tombstone so they can clear local copies; audit history stays.';

  @override
  String get btnConfirmDelete => 'Delete forever';

  @override
  String get btnConfirmRevoke => 'Withdraw';

  @override
  String get navApiKeys => 'API Keys';

  @override
  String get titleApiKeys => 'API Keys';

  @override
  String get descApiKeys =>
      'Manage personal and machine API keys for client and SDK access.';

  @override
  String get btnCreateApiKey => 'Create key';

  @override
  String get labelKeyName => 'Key name / label';

  @override
  String get labelExpiresInDays => 'Expiration in days (optional)';

  @override
  String get labelExpiresNever => 'Never expires';

  @override
  String get labelKeyPrefix => 'Token prefix';

  @override
  String get labelCreatedAt => 'Created at';

  @override
  String get labelExpiresAt => 'Expires at';

  @override
  String get labelLastUsedAt => 'Last used';

  @override
  String get labelNeverUsed => 'Never';

  @override
  String get btnRotateKey => 'Rotate key';

  @override
  String get btnDeleteKey => 'Revoke & Delete';

  @override
  String get dialogCreatedKeyTitle => 'Save your API Key';

  @override
  String get dialogCreatedKeyWarning =>
      'Copy this key now. For your security, you will not be able to see it again!';

  @override
  String get btnCopyKey => 'Copy to clipboard';

  @override
  String get copiedToClipboard => 'Copied to clipboard!';

  @override
  String get confirmRotateKeyTitle => 'Rotate this API Key?';

  @override
  String get confirmRotateKeyBody =>
      'The existing key will immediately stop working and a new key will be generated.';

  @override
  String get confirmDeleteKeyTitle => 'Delete and revoke this API Key?';

  @override
  String get confirmDeleteKeyBody =>
      'Any client or SDK using this API key will be permanently unauthorized.';

  @override
  String get emptyApiKeys =>
      'No API keys found. Click \'Create key\' to issue one.';

  @override
  String get tabPersonalKeys => 'Personal keys';

  @override
  String get tabSystemKeys => 'All system keys (Admin)';
}
