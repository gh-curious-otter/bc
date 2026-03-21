/**
 * useOptimizedImage Hook
 *
 * Simplifies image optimization implementation across components
 * Provides responsive srcset, lazy loading, and WebP support
 */

import { useMemo } from "react";
import {
  generateSrcSet,
  loadingStrategies,
} from "@/lib/image-optimization";

interface UseOptimizedImageProps {
  basePath: string; // e.g. "/images/hero"
  format?: "jpg" | "png"; // Fallback format
  loading?: "eager" | "lazy" | "auto";
  aspect?: {
    width: number;
    height: number;
  };
  sizes?: string; // Custom sizes query string
}

interface OptimizedImageData {
  webpSrcSet: string;
  jpgSrcSet: string;
  sizes: string;
  loading: "eager" | "lazy";
  decoding: "sync" | "async";
  aspectRatio?: number;
  style?: {
    aspectRatio: string;
  };
}

/**
 * Hook for generating optimized image props
 *
 * Usage:
 * ```tsx
 * const { webpSrcSet, jpgSrcSet, sizes, loading } = useOptimizedImage({
 *   basePath: "/images/hero",
 *   format: "jpg",
 *   loading: "eager",
 * });
 *
 * return (
 *   <picture>
 *     <source srcSet={webpSrcSet} type="image/webp" sizes={sizes} />
 *     <img srcSet={jpgSrcSet} src="/images/hero-1280.jpg" sizes={sizes} loading={loading} alt="" />
 *   </picture>
 * );
 * ```
 */
export function useOptimizedImage({
  basePath,
  format = "jpg",
  loading = "lazy",
  aspect,
  sizes: customSizes,
}: UseOptimizedImageProps): OptimizedImageData {
  return useMemo(() => {
    const strategy = loadingStrategies[loading];

    // Default sizes for responsive images
    const defaultSizes =
      "(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 100vw";

    return {
      webpSrcSet: generateSrcSet(basePath, ".webp"),
      jpgSrcSet: generateSrcSet(basePath, `.${format}`),
      sizes: customSizes || defaultSizes,
      loading: strategy.loading,
      decoding: strategy.decoding,
      aspectRatio: aspect ? aspect.width / aspect.height : undefined,
      style: aspect
        ? {
            aspectRatio: `${aspect.width} / ${aspect.height}`,
          }
        : undefined,
    };
  }, [basePath, format, loading, aspect, customSizes]);
}

/**
 * Hook for batch image asset management
 *
 * Usage:
 * ```tsx
 * const images = useImageAssets({
 *   hero: { basePath: "/images/hero", loading: "eager" },
 *   content: { basePath: "/images/content", loading: "lazy" },
 * });
 * ```
 */
export function useImageAssets(
  assets: Record<string, UseOptimizedImageProps>
): Record<string, OptimizedImageData> {
  return useMemo(() => {
    const result: Record<string, OptimizedImageData> = {};
    for (const [key, props] of Object.entries(assets)) {
      result[key] = {
        webpSrcSet: generateSrcSet(props.basePath, ".webp"),
        jpgSrcSet: generateSrcSet(props.basePath, `.${props.format || "jpg"}`),
        sizes:
          props.sizes ||
          "(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 100vw",
        loading: loadingStrategies[props.loading || "lazy"].loading,
        decoding: loadingStrategies[props.loading || "lazy"].decoding,
      };
    }
    return result;
  }, [assets]);
}
