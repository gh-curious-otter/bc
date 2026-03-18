"use client";

import { CSSProperties, ImgHTMLAttributes } from "react";

interface OptimizedImageProps extends Omit<ImgHTMLAttributes<HTMLImageElement>, 'src'> {
  src: string;
  alt: string;
  webpSrc?: string;
  sources?: Array<{
    srcSet: string;
    type?: string;
    media?: string;
  }>;
  containerClassName?: string;
  containerStyle?: CSSProperties;
  loading?: "lazy" | "eager";
  decoding?: "async" | "sync" | "auto";
}

/**
 * OptimizedImage Component
 *
 * Handles image optimization with:
 * - WebP format with fallback support
 * - Responsive srcset for different screen sizes
 * - Native lazy loading
 * - Async decoding for better performance
 *
 * Usage:
 * ```tsx
 * <OptimizedImage
 *   src="/images/hero.jpg"
 *   webpSrc="/images/hero.webp"
 *   alt="Hero section"
 *   loading="lazy"
 *   className="w-full h-auto"
 * />
 * ```
 */
export function OptimizedImage({
  src,
  alt,
  webpSrc,
  sources = [],
  containerClassName = "",
  containerStyle,
  loading = "lazy",
  decoding = "async",
  className = "",
  style,
  ...rest
}: OptimizedImageProps) {
  return (
    <picture style={containerStyle} className={containerClassName}>
      {/* WebP source with responsive sizes */}
      {webpSrc && (
        <source
          srcSet={webpSrc}
          type="image/webp"
          sizes={rest.sizes}
        />
      )}

      {/* Additional responsive sources */}
      {sources.map((source, idx) => (
        <source
          key={idx}
          srcSet={source.srcSet}
          type={source.type}
          media={source.media}
          sizes={rest.sizes}
        />
      ))}

      {/* Fallback PNG/JPG image */}
      <img
        src={src}
        alt={alt}
        loading={loading}
        decoding={decoding}
        className={className}
        style={style}
        {...rest}
      />
    </picture>
  );
}

/**
 * Image configuration helper for creating srcset strings
 * Useful for responsive images with multiple sizes
 */
export function createResponsiveSrcSet(
  basePath: string,
  format: string = "jpg",
  sizes: number[] = [320, 640, 960, 1280, 1920]
): string {
  return sizes
    .map((size) => `${basePath.replace(/\.[^.]+$/, "")}-${size}w.${format} ${size}w`)
    .join(", ");
}

/**
 * Image optimization guidelines for developers
 *
 * BEFORE ADDING IMAGES:
 * 1. Optimize to WebP format (smallest file size)
 * 2. Provide JPEG/PNG fallback
 * 3. Create multiple sizes: 320w, 640w, 960w, 1280w, 1920w
 * 4. Target: < 200KB per image for optimal Core Web Vitals
 *
 * TOOLS:
 * - Convert to WebP: `cwebp image.jpg -o image.webp -q 85`
 * - Batch resize: ImageMagick, Sharp, or online tools
 * - Validate size: `ls -lh image.*`
 *
 * PERFORMANCE CHECKLIST:
 * ✓ All images < 200KB
 * ✓ WebP with fallback
 * ✓ Responsive srcset defined
 * ✓ loading="lazy" on non-hero images
 * ✓ Proper alt text for accessibility
 */

interface ImageOptimizationConfig {
  maxFileSize: number; // bytes
  formats: string[];
  responsiveSizes: number[];
  compressionQuality: number;
}

export const imageOptimizationConfig: ImageOptimizationConfig = {
  maxFileSize: 200 * 1024, // 200KB
  formats: ["webp", "jpeg", "png"],
  responsiveSizes: [320, 640, 960, 1280, 1920],
  compressionQuality: 85,
};
