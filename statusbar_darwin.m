#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

static NSStatusItem *muxStatusItem;

static void muxStartOnMain(void *context) {
    (void)context;
    if (muxStatusItem != nil) return;
    muxStatusItem = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength] retain];
    muxStatusItem.button.title = @"Mux";
    muxStatusItem.button.toolTip = @"Mux 账号用量";
}

static void muxSetTitleOnMain(void *context) {
    char *title = context;
    if (muxStatusItem != nil) {
        muxStatusItem.button.title = title ? [NSString stringWithUTF8String:title] : @"Mux";
    }
    free(title);
}

static void muxStopOnMain(void *context) {
    (void)context;
    if (muxStatusItem == nil) return;
    NSStatusItem *item = muxStatusItem;
    muxStatusItem = nil;
    [[NSStatusBar systemStatusBar] removeStatusItem:item];
    [item release];
}

void muxStatusBarStart(void) {
    dispatch_async_f(dispatch_get_main_queue(), NULL, muxStartOnMain);
}

void muxStatusBarSetTitle(const char *title) {
    char *copy = title ? strdup(title) : NULL;
    dispatch_async_f(dispatch_get_main_queue(), copy, muxSetTitleOnMain);
}

void muxStatusBarStop(void) {
    dispatch_async_f(dispatch_get_main_queue(), NULL, muxStopOnMain);
}
