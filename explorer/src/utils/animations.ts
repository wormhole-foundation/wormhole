import { gsap } from "gsap";

export const fadeInGsap = (target:any)=>{
  gsap.from(target.current, {
    opacity: 0,
    y: 100,
    duration: 2,
    ease: "Power3.easeOut",
    clearProps: "all",
    scrollTrigger: {
      trigger: target.current,
      start: "bottom: 70%",
    },
  });
}

export const paralaxGsap = (target:any, y:number, start:string) => {
  gsap.from(target.current, {
    ease: "Power3.easeOut",
    y: y,
    scrollTrigger: {
      trigger: target.current,
      start: start,
      scrub: 1,
    },
});
}

export const animateSwirl = (target:any) =>{
  gsap.from(target.current, {
    scale: 1.2,
    duration: 10,
    delay: .2,
    rotation: 5,
    ease: "Power3.easeOut",
  })
}
