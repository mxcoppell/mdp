// Standalone tool to generate the dock icon as a PNG file.
// Usage: clang -framework Cocoa -framework CoreGraphics -framework CoreText -o /tmp/gen-icon scripts/gen-icon.m && /tmp/gen-icon assets/dock-icon.png
#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#import <CoreText/CoreText.h>

int main(int argc, const char *argv[]) {
    if (argc < 2) {
        fprintf(stderr, "usage: gen-icon <output.png>\n");
        return 1;
    }
    @autoreleasepool {
        int size = 512;
        CGColorSpaceRef space = CGColorSpaceCreateDeviceRGB();
        CGContextRef ctx = CGBitmapContextCreate(NULL, size, size, 8, size * 4, space,
            (CGBitmapInfo)kCGImageAlphaPremultipliedLast);
        CGColorSpaceRelease(space);
        if (!ctx) return 1;

        CGFloat radius = size * 0.22;
        CGMutablePathRef path = CGPathCreateMutable();
        CGPathMoveToPoint(path, NULL, radius, 0);
        CGPathAddLineToPoint(path, NULL, size - radius, 0);
        CGPathAddArc(path, NULL, size - radius, radius, radius, -M_PI_2, 0, false);
        CGPathAddLineToPoint(path, NULL, size, size - radius);
        CGPathAddArc(path, NULL, size - radius, size - radius, radius, 0, M_PI_2, false);
        CGPathAddLineToPoint(path, NULL, radius, size);
        CGPathAddArc(path, NULL, radius, size - radius, radius, M_PI_2, M_PI, false);
        CGPathAddLineToPoint(path, NULL, 0, radius);
        CGPathAddArc(path, NULL, radius, radius, radius, M_PI, M_PI + M_PI_2, false);
        CGPathCloseSubpath(path);

        CGContextSaveGState(ctx);
        CGContextAddPath(ctx, path);
        CGContextClip(ctx);

        CGFloat colors[] = {
            0.14, 0.15, 0.17, 1.0,
            0.20, 0.21, 0.24, 1.0,
        };
        CGColorSpaceRef gradSpace = CGColorSpaceCreateDeviceRGB();
        CGGradientRef gradient = CGGradientCreateWithColorComponents(gradSpace, colors, NULL, 2);
        CGContextDrawLinearGradient(ctx, gradient, CGPointMake(0, size), CGPointMake(0, 0), 0);
        CGGradientRelease(gradient);
        CGColorSpaceRelease(gradSpace);
        CGContextRestoreGState(ctx);

        CTFontRef mdFont = CTFontCreateWithName(CFSTR("HelveticaNeue-Bold"), size * 0.32, NULL);
        NSDictionary *mdAttrs = @{
            (id)kCTFontAttributeName: (__bridge id)mdFont,
            (id)kCTForegroundColorAttributeName: (__bridge id)[[NSColor whiteColor] CGColor],
        };
        NSAttributedString *mdStr = [[NSAttributedString alloc] initWithString:@"MD" attributes:mdAttrs];
        CTLineRef mdLine = CTLineCreateWithAttributedString((__bridge CFAttributedStringRef)mdStr);
        CGRect mdBounds = CTLineGetBoundsWithOptions(mdLine, 0);
        CGFloat mdX = (size - mdBounds.size.width) / 2 - mdBounds.origin.x;
        CGFloat mdY = size * 0.42;
        CGContextSetTextPosition(ctx, mdX, mdY);
        CTLineDraw(mdLine, ctx);
        CFRelease(mdLine);
        CFRelease(mdFont);

        CTFontRef promptFont = CTFontCreateWithName(CFSTR("Menlo-Bold"), size * 0.22, NULL);
        CGFloat accentColor[] = {0.4, 0.75, 0.95, 1.0};
        CGColorSpaceRef accentSpace = CGColorSpaceCreateDeviceRGB();
        CGColorRef accent = CGColorCreate(accentSpace, accentColor);
        CGColorSpaceRelease(accentSpace);
        NSDictionary *promptAttrs = @{
            (id)kCTFontAttributeName: (__bridge id)promptFont,
            (id)kCTForegroundColorAttributeName: (__bridge id)accent,
        };
        NSAttributedString *promptStr = [[NSAttributedString alloc] initWithString:@">_" attributes:promptAttrs];
        CTLineRef promptLine = CTLineCreateWithAttributedString((__bridge CFAttributedStringRef)promptStr);
        CGRect promptBounds = CTLineGetBoundsWithOptions(promptLine, 0);
        CGFloat promptX = (size - promptBounds.size.width) / 2 - promptBounds.origin.x;
        CGFloat promptY = size * 0.14;
        CGContextSetTextPosition(ctx, promptX, promptY);
        CTLineDraw(promptLine, ctx);
        CFRelease(promptLine);
        CFRelease(promptFont);
        CGColorRelease(accent);

        CGImageRef cgImage = CGBitmapContextCreateImage(ctx);
        CGContextRelease(ctx);
        CGPathRelease(path);

        if (!cgImage) return 1;

        NSString *outPath = [NSString stringWithUTF8String:argv[1]];
        NSURL *url = [NSURL fileURLWithPath:outPath];
        CGImageDestinationRef dest = CGImageDestinationCreateWithURL((__bridge CFURLRef)url, kUTTypePNG, 1, NULL);
        CGImageDestinationAddImage(dest, cgImage, NULL);
        CGImageDestinationFinalize(dest);
        CFRelease(dest);
        CGImageRelease(cgImage);
    }
    return 0;
}
