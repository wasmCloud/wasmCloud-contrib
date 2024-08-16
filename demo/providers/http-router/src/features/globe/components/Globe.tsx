import {useSatellites} from '@/features/globe/hooks/useSatellites';
import {useTheme} from '@/features/theme/hooks/useTheme';
import React, {ComponentProps} from 'react';
import ReactGlobeGL from 'react-globe.gl';
import {PerspectiveCamera} from 'three';

type GlobeMethods = ComponentProps<typeof ReactGlobeGL>['ref'] extends
  | React.MutableRefObject<infer T>
  | undefined
  ? Exclude<T, undefined>
  : never;

function Globe() {
  const [theme] = useTheme();

  const container = React.useRef<HTMLDivElement>(null);
  const inner = React.useRef<HTMLDivElement>(null);
  const globeEl = React.useRef<GlobeMethods>();
  const [globeRadius, setGlobeRadius] = React.useState<number>();
  const [size, setSize] = React.useState<{
    width: number;
    height: number;
  }>();
  const {objectsData, satObject} = useSatellites(globeRadius);

  const getDimensions = React.useCallback(() => {
    const containerEl = container.current;
    if (!containerEl) {
      return {cY: 0, cX: 0, cW: 0, cH: 0, sW: 0, sH: 0};
    }
    const {top: cY, left: cX, width: cW, height: cH} = containerEl.getBoundingClientRect();
    const sW = document.documentElement.clientWidth;
    const sH = document.documentElement.clientHeight;
    return {cY, cX, cW, cH, sW, sH};
  }, []);

  React.useEffect(() => {
    const containerEl = container.current;
    const innerEl = inner.current;
    if (!containerEl || !innerEl || !globeEl.current) return;
    const camera = globeEl.current?.camera() as PerspectiveCamera;

    const calculateSize = () => {
      window.requestAnimationFrame(() => {
        // Reset margin so that it doesn't affect the size of the container
        innerEl.style.margin = '-99999px';

        // Calculate container and screen dimensions
        const {cY, cX, cW, cH, sW, sH} = getDimensions();

        // Calculate margins to offset the globe to the size of the container
        const mt = cY * -1;
        const mr = (sW - cW - cX) * -1;
        const mb = (sH - cH - cY) * -1;
        const ml = cX * -1;

        // Apply margins and set size
        innerEl.style.margin = `${mt}px ${mr}px ${mb}px ${ml}px`;
        setSize({width: sW, height: sH});

        // TODO: Make the camera fit the container, rather than the whole screen
        // offset the camera to the center of the container based on its position in the screen

        // const vw = sW;
        // const vh = sH;
        // const vx = cX - sW + cW;
        // const vy = cY - sH + cH;

        // camera.aspect = cW / cH;
        // camera.setViewOffset(cW, cH, vx, vy, cW, cH);
      });
    };

    calculateSize();
    window.addEventListener('resize', calculateSize);

    return () => {
      window.removeEventListener('resize', calculateSize);
    };
  }, []);

  React.useEffect(() => {
    if (!globeEl.current) return;
    setGlobeRadius(globeEl.current.getGlobeRadius());
    globeEl.current.pointOfView({altitude: 3.5});
  }, []);

  const {width, height} = size || {};

  return (
    <div ref={container} className="grow h-[30vh] -z-10">
      <div ref={inner}>
        <ReactGlobeGL
          ref={globeEl}
          objectsData={objectsData}
          objectThreeObject={satObject}
          globeImageUrl="//unpkg.com/three-globe/example/img/earth-dark.jpg"
          bumpImageUrl="//unpkg.com/three-globe/example/img/earth-topology.png"
          backgroundImageUrl="//unpkg.com/three-globe/example/img/night-sky.png"
          width={width}
          height={height}
        />
      </div>
    </div>
  );
}

export {Globe};
