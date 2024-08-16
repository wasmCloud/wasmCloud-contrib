import React from 'react';
import * as satellite from 'satellite.js';
import * as THREE from 'three';

const EARTH_RADIUS_KM = 6371; // km
const SAT_SIZE = 80; // km
const TIME_STEP = 3 * 1000; // per frame

function useSatellites(globeRadius: number | undefined) {
  const [satData, setSatData] = React.useState<{satrec: satellite.SatRec; name: string}[]>();
  const [time, setTime] = React.useState(new Date());

  React.useEffect(() => {
    // time ticker
    (function frameTicker() {
      requestAnimationFrame(frameTicker);
      setTime((time) => new Date(+time + TIME_STEP));
    })();
  }, []);

  React.useEffect(() => {
    // load satellite data
    fetch('//unpkg.com/globe.gl/example/datasets/space-track-leo.txt')
      .then((r) => r.text())
      .then((rawData) => {
        const tleData = rawData
          .replace(/\r/g, '')
          .split(/\n(?=[^12])/)
          .filter((d) => d)
          .map((tle) => tle.split('\n'));
        const satData = tleData
          .map(([name, ...tle]) => ({
            satrec: satellite.twoline2satrec(tle[0], tle[1]),
            name: name.trim().replace(/^0 /, ''),
          }))
          // exclude those that can't be propagated
          .filter((d) => !!satellite.propagate(d.satrec, new Date()).position)
          .slice(0, 1500);

        setSatData(satData);
      });
  }, []);

  const objectsData = React.useMemo(() => {
    if (!satData) return [];

    // Update satellite positions
    const gmst = satellite.gstime(time);
    return satData.map((d) => {
      const eci = satellite.propagate(d.satrec, time);
      if (eci.position && typeof eci.position !== 'boolean') {
        const gdPos = satellite.eciToGeodetic(eci.position, gmst);
        // @ts-expect-error -- radiansToDegrees is not in the types but it is a real function, a real human bean
        const lat = satellite.radiansToDegrees(gdPos.latitude);
        // @ts-expect-error -- radiansToDegrees is not in the types but it is a real function, a real human bean
        const lng = satellite.radiansToDegrees(gdPos.longitude);
        const alt = gdPos.height / EARTH_RADIUS_KM;
        return {...d, lat, lng, alt};
      }
      return d;
    });
  }, [satData, time]);

  const satObject = React.useMemo(() => {
    if (!globeRadius) return undefined;

    const satGeometry = new THREE.OctahedronGeometry(
      (SAT_SIZE * globeRadius) / EARTH_RADIUS_KM / 2,
      0,
    );
    const satMaterial = new THREE.MeshLambertMaterial({
      color: 'palegreen',
      transparent: true,
      opacity: 0.7,
    });
    return new THREE.Mesh(satGeometry, satMaterial);
  }, [globeRadius]);

  return {objectsData, satObject};
}

export {useSatellites};
