#import <Cocoa/Cocoa.h>

static NSStatusItem *muxStatusItem;

void muxStatusBarStart(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (muxStatusItem != nil) return;
        muxStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        muxStatusItem.button.title = @"Mux";
        muxStatusItem.button.toolTip = @"Mux 账号用量";
    });
}

void muxStatusBarSetTitle(const char *title) {
    NSString *value = title ? [NSString stringWithUTF8String:title] : @"Mux";
    dispatch_async(dispatch_get_main_queue(), ^{
        if (muxStatusItem != nil) muxStatusItem.button.title = value;
    });
}

void muxStatusBarStop(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (muxStatusItem != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:muxStatusItem];
            muxStatusItem = nil;
        }
    });
}
