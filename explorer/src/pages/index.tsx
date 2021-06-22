import React, { useRef, useEffect } from 'react';
import { Typography } from 'antd'
const { Title } = Typography
import { useIntl } from 'gatsby-plugin-intl';

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';

import { ReactComponent as EthereumIcon } from '~/icons/ethereum.svg';
import { ReactComponent as SolanaIcon } from '~/icons/solana.svg';
import { ReactComponent as TerraIcon } from '~/icons/terra.svg';
import { ReactComponent as BinanceChainIcon } from '~/icons/binancechain.svg';

const headerHeight = 64
const footerHeight = 88
const contentHeight = `calc(100vh - (${headerHeight}px + ${footerHeight}px))`

const Index = () => {
  const intl = useIntl()
  const canvasRef = useRef<HTMLCanvasElement>(null);


  useEffect(() => {
    if (canvasRef.current) {

      // (maybe) TODO: refactor this away from es5?
      // maybe not, as this will be replaced by Breakout (design firm) designs.
  
      // pulled this animation from here https://codepen.io/lans/pen/WQGZoZ
      // Global Animation Setting
      window.requestAnimFrame =
        window.requestAnimationFrame ||
        window.webkitRequestAnimationFrame ||
        window.mozRequestAnimationFrame ||
        window.oRequestAnimationFrame ||
        window.msRequestAnimationFrame ||
        function (callback: any) {
          window.setTimeout(callback, 1000 / 60);
        };

      // Global Canvas Setting
      let canvas = canvasRef.current // document.getElementById('particle');
      let ctx = canvas.getContext('2d');
      if (ctx != null) {

        const resizeCanvas = () => {
          canvas.width = window.innerWidth
          canvas.height = (window.innerHeight - (headerHeight + footerHeight))
          emitter = new Emitter(canvas.width / 2, canvas.height / 2);
        }
        window.addEventListener('resize', resizeCanvas, false);

        canvas.width = window.innerWidth
        canvas.height = (window.innerHeight - (headerHeight + footerHeight))

        // Particles Around the Parent
        const Particle = function (x: number, y: number, dist: number, rbgStr: string) {
          this.angle = Math.random() * 2 * Math.PI;
          this.radius = Math.random();
          this.opacity = (Math.random() * 5 + 2) / 10;
          this.distance = (1 / this.opacity) * dist;
          this.speed = this.distance * 0.00009

          this.position = {
            x: x + this.distance * Math.cos(this.angle),
            y: y + this.distance * Math.sin(this.angle)
          };

          this.draw = function () {
            if (!ctx) return
            ctx.fillStyle = `rgba(${rbgStr},${this.opacity})`
            ctx.beginPath();
            ctx.arc(this.position.x, this.position.y, this.radius, 0, Math.PI * 2, false);
            ctx.fill();
            ctx.closePath();
          }
          this.update = function () {
            this.angle += this.speed;
            this.position = {
              x: x + this.distance * Math.cos(this.angle),
              y: y + this.distance * Math.sin(this.angle)
            };
            this.draw();
          }
        }

        const Emitter = function (x: number, y: number) {
          this.position = { x: x, y: y };
          this.radius = 60;
          this.count = 4000;
          this.particles = [];

          for (var i = 0; i < this.count; i++) {
            this.particles.push(new Particle(this.position.x, this.position.y, this.radius, "255,233,31"));
            this.particles.push(new Particle(this.position.x, this.position.y, this.radius, "255,110,253"));
            this.particles.push(new Particle(this.position.x, this.position.y, this.radius, "166,128,255"));
            this.particles.push(new Particle(this.position.x, this.position.y, this.radius, "128,232,255"));
          }
        }

        Emitter.prototype = {
          draw: function () {
            if (!ctx) return
            ctx.fillStyle = "rgba(0,0,0,1)";
            ctx.beginPath();
            ctx.arc(this.position.x, this.position.y, this.radius, 0, Math.PI * 2, false);
            ctx.fill();
            ctx.closePath();
          },
          update: function () {
            for (var i = 0; i < this.count; i++) {
              this.particles[i].update();
            }
            this.draw();
          }
        }

        let emitter = new Emitter(canvas.width / 2, canvas.height / 2);

        const loop = () => {
          if (!ctx) return
          ctx.clearRect(0, 0, canvas.width, canvas.height);
          ctx.canvas.width = window.innerWidth
          ctx.canvas.height = (window.innerHeight - (headerHeight + footerHeight))
          emitter.update();
          window.requestAnimFrame(loop);
        }

        loop();
      }
    }
  }, []);
  const iconStyles = { width: 100, margin: '0 4px' }
  const iconWithCaption = (IconEle: React.ElementType, caption: string, figureStyle: object = {}) => {
    return <figure style={{...iconStyles, ...figureStyle}}>
      <IconEle />
      <figcaption style={{textAlign: 'center'}}>{caption}</figcaption>
    </figure>
  }
  return (
    <Layout>
      <SEO 
        title={intl.formatMessage({ id: 'homepage.title' })}
        description={intl.formatMessage({ id: 'homepage.description' })}
      />
      <div style={{ position: 'relative', height: contentHeight, width:"100vw" }}>
        <div style={{ position: 'absolute', top: '5vh', zIndex: 100, width: '100%', textAlign: 'center'}}>

          <Title level={1}>{intl.formatMessage({ id: 'homepage.description' })}</Title>
          <Title level={2}>{intl.formatMessage({ id: 'homepage.subtext' })}</Title>
        </div>
        <div style={{ position: 'absolute', bottom: '5vh', zIndex: 100, width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-evenly', alignContent: 'flex-end', 
                        alignItems: 'center', margin: '0 8px'}}>
            {iconWithCaption(EthereumIcon, 'Ethereum', {width: 70})}
            {iconWithCaption(TerraIcon, 'Terra')}
            {iconWithCaption(BinanceChainIcon, 'Binance Smart Chain')}
            {iconWithCaption(SolanaIcon, 'Solana')}
          </div>
        </div>
        <canvas ref={canvasRef} height={contentHeight} width="100vw" style={{ position: 'relative', zIndex: 99 }} />

      </div>
    </Layout>
  );
};

export default Index
