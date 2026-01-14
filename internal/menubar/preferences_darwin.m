#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>

@interface PreferencesController : NSObject <NSWindowDelegate>
@property (strong, nonatomic) NSWindow *window;
@property (strong, nonatomic) NSTextField *daemonAddrField;
@property (strong, nonatomic) NSTextField *syncIntervalField;
@property (strong, nonatomic) NSButton *ocrCheckbox;
@property (strong, nonatomic) NSTextField *daemonConfigField;
@property (nonatomic) BOOL saved;
@property (copy, nonatomic) NSString *daemonAddr;
@property (copy, nonatomic) NSString *syncInterval;
@property (nonatomic) BOOL ocrEnabled;
@property (copy, nonatomic) NSString *daemonConfigFile;
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
        self.saved = NO;
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
    [warningLabel setBackgroundColor:[NSColor colorWithRed:1.0 green:0.95 blue:0.8 alpha:1.0]];
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
    self.saved = YES;
    [self.window close];
}

- (void)cancelClicked:(id)sender {
    self.saved = NO;
    [self.window close];
}

- (void)showModal {
    [self.window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

@end

// C interface for Go
typedef struct {
    const char *daemonAddr;
    const char *syncInterval;
    int ocrEnabled;
    const char *daemonConfigFile;
    int saved;
} PreferencesResult;

void *createPreferencesController(const char *daemonAddr,
                                  const char *syncInterval,
                                  int ocrEnabled,
                                  const char *daemonConfigFile) {
    @autoreleasepool {
        NSString *nsAddr = [NSString stringWithUTF8String:daemonAddr];
        NSString *nsInterval = [NSString stringWithUTF8String:syncInterval];
        NSString *nsConfig = [NSString stringWithUTF8String:daemonConfigFile];

        PreferencesController *controller = [[PreferencesController alloc]
            initWithDaemonAddr:nsAddr
            syncInterval:nsInterval
            ocrEnabled:(ocrEnabled != 0)
            daemonConfigFile:nsConfig];

        [controller retain];
        return (__bridge void *)controller;
    }
}

void showPreferencesWindow(void *controller) {
    @autoreleasepool {
        PreferencesController *ctrl = (__bridge PreferencesController *)controller;
        [ctrl showModal];

        // Run modal loop
        NSModalSession session = [NSApp beginModalSessionForWindow:ctrl.window];
        while ([NSApp runModalSession:session] == NSModalResponseContinue) {
            [[NSRunLoop currentRunLoop] runMode:NSDefaultRunLoopMode
                                     beforeDate:[NSDate distantFuture]];
            if (![[ctrl window] isVisible]) {
                break;
            }
        }
        [NSApp endModalSession:session];
    }
}

PreferencesResult getPreferencesResult(void *controller) {
    PreferencesResult result;
    @autoreleasepool {
        PreferencesController *ctrl = (__bridge PreferencesController *)controller;

        result.daemonAddr = strdup([ctrl.daemonAddr UTF8String]);
        result.syncInterval = strdup([ctrl.syncInterval UTF8String]);
        result.ocrEnabled = ctrl.ocrEnabled ? 1 : 0;
        result.daemonConfigFile = strdup([ctrl.daemonConfigFile UTF8String]);
        result.saved = ctrl.saved ? 1 : 0;
    }
    return result;
}

void releasePreferencesController(void *controller) {
    @autoreleasepool {
        PreferencesController *ctrl = (__bridge PreferencesController *)controller;
        [ctrl release];
    }
}
