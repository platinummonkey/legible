#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>

// Forward declare the Go callback function
extern void preferencesGoCallback(const char *daemonAddr,
                                  const char *syncInterval,
                                  int ocrEnabled,
                                  const char *daemonConfigFile,
                                  void *context);

@interface PreferencesController : NSObject <NSWindowDelegate>
@property (strong, nonatomic) NSWindow *window;
@property (strong, nonatomic) NSTextField *daemonAddrField;
@property (strong, nonatomic) NSTextField *syncIntervalField;
@property (strong, nonatomic) NSButton *ocrCheckbox;
@property (strong, nonatomic) NSTextField *daemonConfigField;
@property (copy, nonatomic) NSString *daemonAddr;
@property (copy, nonatomic) NSString *syncInterval;
@property (nonatomic) BOOL ocrEnabled;
@property (copy, nonatomic) NSString *daemonConfigFile;
@property (nonatomic) void *callbackContext;
@end

@implementation PreferencesController

- (instancetype)initWithDaemonAddr:(NSString *)daemonAddr
                      syncInterval:(NSString *)syncInterval
                        ocrEnabled:(BOOL)ocrEnabled
                  daemonConfigFile:(NSString *)daemonConfigFile {
    self = [super init];
    if (self) {
        self.daemonAddr = daemonAddr;
        self.syncInterval = syncInterval;
        self.ocrEnabled = ocrEnabled;
        self.daemonConfigFile = daemonConfigFile;
        [self createWindow];
    }
    return self;
}

- (void)createWindow {
    // Create window
    NSRect frame = NSMakeRect(0, 0, 500, 320);
    NSWindowStyleMask styleMask = NSWindowStyleMaskTitled |
                                  NSWindowStyleMaskClosable |
                                  NSWindowStyleMaskMiniaturizable;

    self.window = [[NSWindow alloc] initWithContentRect:frame
                                              styleMask:styleMask
                                                backing:NSBackingStoreBuffered
                                                  defer:NO];
    [self.window setTitle:@"Legible Preferences"];
    [self.window setDelegate:self];
    [self.window center];

    // Enable dark mode support
    if (@available(macOS 10.14, *)) {
        [self.window setAppearance:nil]; // Use system appearance
    }

    // Create content view
    NSView *contentView = [[NSView alloc] initWithFrame:frame];
    [self.window setContentView:contentView];

    CGFloat y = frame.size.height - 40;
    CGFloat labelWidth = 140;
    CGFloat fieldWidth = 320;
    CGFloat rowHeight = 50;

    // Warning label
    NSTextField *warningLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(20, y, 460, 30)];
    [warningLabel setStringValue:@"⚠️  Changes require restarting the menu bar app"];
    [warningLabel setBezeled:NO];
    [warningLabel setDrawsBackground:YES];
    if (@available(macOS 10.14, *)) {
        [warningLabel setBackgroundColor:[NSColor.systemOrangeColor colorWithAlphaComponent:0.2]];
        [warningLabel setTextColor:[NSColor labelColor]];
    } else {
        [warningLabel setBackgroundColor:[NSColor colorWithRed:1.0 green:0.95 blue:0.8 alpha:1.0]];
    }
    [warningLabel setEditable:NO];
    [warningLabel setAlignment:NSTextAlignmentCenter];
    [contentView addSubview:warningLabel];

    y -= rowHeight + 10;

    // Daemon Address
    NSTextField *daemonAddrLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(20, y, labelWidth, 24)];
    [daemonAddrLabel setStringValue:@"Daemon Address:"];
    [daemonAddrLabel setBezeled:NO];
    [daemonAddrLabel setDrawsBackground:NO];
    [daemonAddrLabel setEditable:NO];
    [daemonAddrLabel setSelectable:NO];
    if (@available(macOS 10.10, *)) {
        [daemonAddrLabel setTextColor:[NSColor labelColor]];
    }
    [contentView addSubview:daemonAddrLabel];

    self.daemonAddrField = [[NSTextField alloc] initWithFrame:NSMakeRect(160, y, fieldWidth, 24)];
    [self.daemonAddrField setStringValue:self.daemonAddr];
    [self.daemonAddrField setPlaceholderString:@"http://localhost:8080"];
    [contentView addSubview:self.daemonAddrField];

    y -= rowHeight;

    // Sync Interval
    NSTextField *syncIntervalLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(20, y, labelWidth, 24)];
    [syncIntervalLabel setStringValue:@"Sync Interval:"];
    [syncIntervalLabel setBezeled:NO];
    [syncIntervalLabel setDrawsBackground:NO];
    [syncIntervalLabel setEditable:NO];
    [syncIntervalLabel setSelectable:NO];
    if (@available(macOS 10.10, *)) {
        [syncIntervalLabel setTextColor:[NSColor labelColor]];
    }
    [contentView addSubview:syncIntervalLabel];

    self.syncIntervalField = [[NSTextField alloc] initWithFrame:NSMakeRect(160, y, fieldWidth, 24)];
    [self.syncIntervalField setStringValue:self.syncInterval];
    [self.syncIntervalField setPlaceholderString:@"30m, 1h, 2h"];
    [contentView addSubview:self.syncIntervalField];

    y -= rowHeight;

    // OCR Checkbox
    NSTextField *ocrLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(20, y, labelWidth, 24)];
    [ocrLabel setStringValue:@"OCR Processing:"];
    [ocrLabel setBezeled:NO];
    [ocrLabel setDrawsBackground:NO];
    [ocrLabel setEditable:NO];
    [ocrLabel setSelectable:NO];
    if (@available(macOS 10.10, *)) {
        [ocrLabel setTextColor:[NSColor labelColor]];
    }
    [contentView addSubview:ocrLabel];

    self.ocrCheckbox = [[NSButton alloc] initWithFrame:NSMakeRect(160, y, fieldWidth, 24)];
    [self.ocrCheckbox setButtonType:NSButtonTypeSwitch];
    [self.ocrCheckbox setTitle:@"Enable OCR text layer generation"];
    [self.ocrCheckbox setState:self.ocrEnabled ? NSControlStateValueOn : NSControlStateValueOff];
    [contentView addSubview:self.ocrCheckbox];

    y -= rowHeight;

    // Daemon Config File
    NSTextField *configLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(20, y, labelWidth, 24)];
    [configLabel setStringValue:@"Daemon Config File:"];
    [configLabel setBezeled:NO];
    [configLabel setDrawsBackground:NO];
    [configLabel setEditable:NO];
    [configLabel setSelectable:NO];
    if (@available(macOS 10.10, *)) {
        [configLabel setTextColor:[NSColor labelColor]];
    }
    [contentView addSubview:configLabel];

    self.daemonConfigField = [[NSTextField alloc] initWithFrame:NSMakeRect(160, y, fieldWidth, 24)];
    [self.daemonConfigField setStringValue:self.daemonConfigFile];
    [self.daemonConfigField setPlaceholderString:@"~/.legible.yaml"];
    [contentView addSubview:self.daemonConfigField];

    // Buttons
    NSButton *saveButton = [[NSButton alloc] initWithFrame:NSMakeRect(frame.size.width - 180, 20, 80, 32)];
    [saveButton setTitle:@"Save"];
    [saveButton setBezelStyle:NSBezelStyleRounded];
    [saveButton setKeyEquivalent:@"\r"];
    [saveButton setTarget:self];
    [saveButton setAction:@selector(saveClicked:)];
    [contentView addSubview:saveButton];

    NSButton *cancelButton = [[NSButton alloc] initWithFrame:NSMakeRect(frame.size.width - 90, 20, 80, 32)];
    [cancelButton setTitle:@"Cancel"];
    [cancelButton setBezelStyle:NSBezelStyleRounded];
    [cancelButton setKeyEquivalent:@"\033"];
    [cancelButton setTarget:self];
    [cancelButton setAction:@selector(cancelClicked:)];
    [contentView addSubview:cancelButton];
}

- (void)saveClicked:(id)sender {
    self.daemonAddr = [self.daemonAddrField stringValue];
    self.syncInterval = [self.syncIntervalField stringValue];
    self.ocrEnabled = ([self.ocrCheckbox state] == NSControlStateValueOn);
    self.daemonConfigFile = [self.daemonConfigField stringValue];

    // Call the Go callback directly
    preferencesGoCallback(
        [self.daemonAddr UTF8String],
        [self.syncInterval UTF8String],
        self.ocrEnabled ? 1 : 0,
        [self.daemonConfigFile UTF8String],
        self.callbackContext
    );

    [self.window close];
}

- (void)cancelClicked:(id)sender {
    [self.window close];
}

- (void)show {
    // Already on main thread due to dispatch in showPreferencesWindow
    [self.window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

@end

// C interface for Go
void *createPreferencesController(const char *daemonAddr,
                                  const char *syncInterval,
                                  int ocrEnabled,
                                  const char *daemonConfigFile,
                                  void *context) {
    __block void *result = NULL;

    // MUST run on main thread
    dispatch_sync(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSString *nsAddr = [NSString stringWithUTF8String:daemonAddr];
            NSString *nsInterval = [NSString stringWithUTF8String:syncInterval];
            NSString *nsConfig = [NSString stringWithUTF8String:daemonConfigFile];

            PreferencesController *controller = [[PreferencesController alloc]
                initWithDaemonAddr:nsAddr
                syncInterval:nsInterval
                ocrEnabled:(ocrEnabled != 0)
                daemonConfigFile:nsConfig];

            controller.callbackContext = context;

            [controller retain];
            result = (__bridge void *)controller;
        }
    });

    return result;
}

void showPreferencesWindow(void *controller) {
    // MUST run on main thread
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            PreferencesController *ctrl = (__bridge PreferencesController *)controller;
            [ctrl show];
        }
    });
}

void releasePreferencesController(void *controller) {
    @autoreleasepool {
        PreferencesController *ctrl = (__bridge PreferencesController *)controller;
        [ctrl release];
    }
}
