#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

static NSStatusItem *muxStatusItem;
static NSMenu *muxStatusMenu;

static NSColor *muxUsageColor(double remaining) {
    if (remaining > 0.5) return [NSColor systemGreenColor];
    if (remaining > 0.2) return [NSColor systemYellowColor];
    return [NSColor systemRedColor];
}

static NSImage *muxUsageImage(double usedPercent) {
    double remaining = 1.0 - usedPercent;
    if (remaining < 0.0) remaining = 0.0;
    if (remaining > 1.0) remaining = 1.0;

    NSImage *image = [[[NSImage alloc] initWithSize:NSMakeSize(18.0, 18.0)] autorelease];
    [image lockFocus];
    NSColor *color = muxUsageColor(remaining);
    NSPoint center = NSMakePoint(9.0, 9.0);
    NSBezierPath *track = [NSBezierPath bezierPathWithOvalInRect:NSMakeRect(2.0, 2.0, 14.0, 14.0)];
    track.lineWidth = 2.0;
    [[NSColor blackColor] setStroke];
    [track stroke];
    if (remaining > 0.0) {
        NSBezierPath *progress = [NSBezierPath bezierPath];
        [progress appendBezierPathWithArcWithCenter:center radius:7.0 startAngle:-90.0 endAngle:-90.0 - (360.0 * remaining) clockwise:YES];
        progress.lineWidth = 2.0;
        [color setStroke];
        [progress stroke];
    }
    [image unlockFocus];
    image.template = NO;
    return image;
}

static void muxStartOnMain(void *context) {
    (void)context;
    if (muxStatusItem != nil) return;
    muxStatusItem = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength] retain];
    muxStatusItem.button.title = @"Mux";
    muxStatusItem.button.toolTip = @"Mux 账号用量";
    muxStatusMenu = [[[NSMenu alloc] initWithTitle:@"Mux 账号用量"] autorelease];
    muxStatusItem.menu = muxStatusMenu;
}

static void muxSetTitleOnMain(void *context) {
    char *title = context;
    if (muxStatusItem != nil) {
        muxStatusItem.button.title = title ? [NSString stringWithUTF8String:title] : @"Mux";
    }
    free(title);
}

static void muxSetUsageOnMain(void *context) {
    double usedPercent = *(double *)context;
    free(context);
    if (muxStatusItem == nil) return;
    muxStatusItem.button.image = muxUsageImage(usedPercent);
    muxStatusItem.button.toolTip = [NSString stringWithFormat:@"Mux 账号 7 天用量：已用 %.0f%%，剩余 %.0f%%", usedPercent * 100.0, (1.0 - usedPercent) * 100.0];
}

static void muxSetDetailsOnMain(void *context) {
    char *details = context;
    if (muxStatusMenu != nil) {
        [muxStatusMenu removeAllItems];
        NSString *text = details ? [NSString stringWithUTF8String:details] : @"暂无账号用量数据";
        for (NSString *line in [text componentsSeparatedByString:@"\n"]) {
            NSMenuItem *item = [[[NSMenuItem alloc] initWithTitle:line action:nil keyEquivalent:@""] autorelease];
            item.enabled = NO;
            [muxStatusMenu addItem:item];
        }
    }
    free(details);
}

static void muxStopOnMain(void *context) {
    (void)context;
    if (muxStatusItem == nil) return;
    NSStatusItem *item = muxStatusItem;
    muxStatusItem = nil;
    [[NSStatusBar systemStatusBar] removeStatusItem:item];
    muxStatusMenu = nil;
    [item release];
}

void muxStatusBarStart(void) {
    dispatch_async_f(dispatch_get_main_queue(), NULL, muxStartOnMain);
}

void muxStatusBarSetTitle(const char *title) {
    char *copy = title ? strdup(title) : NULL;
    dispatch_async_f(dispatch_get_main_queue(), copy, muxSetTitleOnMain);
}

void muxStatusBarSetUsage(double usedPercent) {
    double *copy = malloc(sizeof(double));
    *copy = usedPercent;
    dispatch_async_f(dispatch_get_main_queue(), copy, muxSetUsageOnMain);
}

void muxStatusBarSetDetails(const char *details) {
    char *copy = details ? strdup(details) : NULL;
    dispatch_async_f(dispatch_get_main_queue(), copy, muxSetDetailsOnMain);
}

void muxStatusBarStop(void) {
    dispatch_async_f(dispatch_get_main_queue(), NULL, muxStopOnMain);
}
