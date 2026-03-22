import { useCallback, useEffect, useRef, useState } from 'react';
import type { PointerEvent, ReactNode } from 'react';

type CustomScrollAreaProps = {
  className?: string;
  children: ReactNode;
};

type ScrollMetrics = {
  scrollable: boolean;
  thumbHeight: number;
  thumbTop: number;
  maxThumbTop: number;
  maxScrollTop: number;
};

const MIN_THUMB_HEIGHT = 28;

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

export function CustomScrollArea({ className = '', children }: CustomScrollAreaProps) {
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);
  const dragStartRef = useRef<{ pointerId: number; startY: number; startThumbTop: number } | null>(null);
  const metricsRef = useRef<ScrollMetrics>({
    scrollable: false,
    thumbHeight: 0,
    thumbTop: 0,
    maxThumbTop: 0,
    maxScrollTop: 0
  });

  const [metrics, setMetrics] = useState<ScrollMetrics>(metricsRef.current);
  const [isDragging, setIsDragging] = useState(false);

  const updateMetrics = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;

    const { clientHeight, scrollHeight, scrollTop } = viewport;
    const maxScrollTop = Math.max(0, scrollHeight - clientHeight);

    if (maxScrollTop <= 0 || clientHeight <= 0) {
      const nextMetrics: ScrollMetrics = {
        scrollable: false,
        thumbHeight: 0,
        thumbTop: 0,
        maxThumbTop: 0,
        maxScrollTop: 0
      };
      metricsRef.current = nextMetrics;
      setMetrics(nextMetrics);
      return;
    }

    const thumbHeight = clamp((clientHeight / scrollHeight) * clientHeight, MIN_THUMB_HEIGHT, clientHeight);
    const maxThumbTop = Math.max(0, clientHeight - thumbHeight);
    const thumbTop = maxScrollTop > 0 ? (scrollTop / maxScrollTop) * maxThumbTop : 0;
    const nextMetrics: ScrollMetrics = {
      scrollable: true,
      thumbHeight,
      thumbTop,
      maxThumbTop,
      maxScrollTop
    };
    metricsRef.current = nextMetrics;
    setMetrics(nextMetrics);
  }, []);

  useEffect(() => {
    const viewport = viewportRef.current;
    const content = contentRef.current;
    if (!viewport || !content) return;

    updateMetrics();

    const handleScroll = () => {
      updateMetrics();
    };

    viewport.addEventListener('scroll', handleScroll, { passive: true });
    const resizeObserver = new ResizeObserver(() => {
      updateMetrics();
    });
    resizeObserver.observe(viewport);
    resizeObserver.observe(content);
    window.addEventListener('resize', updateMetrics);

    return () => {
      viewport.removeEventListener('scroll', handleScroll);
      resizeObserver.disconnect();
      window.removeEventListener('resize', updateMetrics);
    };
  }, [children, updateMetrics]);

  const setScrollFromThumbTop = useCallback((nextThumbTop: number) => {
    const viewport = viewportRef.current;
    const currentMetrics = metricsRef.current;
    if (!viewport || currentMetrics.maxThumbTop <= 0 || currentMetrics.maxScrollTop <= 0) return;
    const ratio = nextThumbTop / currentMetrics.maxThumbTop;
    viewport.scrollTop = ratio * currentMetrics.maxScrollTop;
  }, []);

  const handleTrackPointerDown = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      if (event.button !== 0) return;
      const target = event.target as HTMLElement;
      if (target.closest('[data-role="custom-scroll-thumb"]')) return;

      const trackRect = event.currentTarget.getBoundingClientRect();
      const nextThumbTop = clamp(event.clientY - trackRect.top - metricsRef.current.thumbHeight / 2, 0, metricsRef.current.maxThumbTop);
      setScrollFromThumbTop(nextThumbTop);
    },
    [setScrollFromThumbTop]
  );

  const handleThumbPointerDown = useCallback((event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    event.preventDefault();
    dragStartRef.current = {
      pointerId: event.pointerId,
      startY: event.clientY,
      startThumbTop: metricsRef.current.thumbTop
    };
    event.currentTarget.setPointerCapture(event.pointerId);
    setIsDragging(true);
  }, []);

  const handleThumbPointerMove = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      const dragStart = dragStartRef.current;
      if (!dragStart || dragStart.pointerId !== event.pointerId) return;
      const deltaY = event.clientY - dragStart.startY;
      const nextThumbTop = clamp(dragStart.startThumbTop + deltaY, 0, metricsRef.current.maxThumbTop);
      setScrollFromThumbTop(nextThumbTop);
    },
    [setScrollFromThumbTop]
  );

  const endDrag = useCallback((event: PointerEvent<HTMLDivElement>) => {
    const dragStart = dragStartRef.current;
    if (!dragStart || dragStart.pointerId !== event.pointerId) return;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    dragStartRef.current = null;
    setIsDragging(false);
  }, []);

  return (
    <div className={`custom-scroll-area ${className}`.trim()}>
      <div ref={viewportRef} className="custom-scroll-area__viewport">
        <div ref={contentRef} className="custom-scroll-area__content">
          {children}
        </div>
      </div>
      {metrics.scrollable && (
        <div className="custom-scroll-area__track" onPointerDown={handleTrackPointerDown}>
          <div
            data-role="custom-scroll-thumb"
            className={`custom-scroll-area__thumb ${isDragging ? 'is-dragging' : ''}`.trim()}
            style={{
              height: `${metrics.thumbHeight}px`,
              transform: `translateY(${metrics.thumbTop}px)`
            }}
            onPointerDown={handleThumbPointerDown}
            onPointerMove={handleThumbPointerMove}
            onPointerUp={endDrag}
            onPointerCancel={endDrag}
          />
        </div>
      )}
    </div>
  );
}
