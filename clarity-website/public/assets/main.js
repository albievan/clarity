// ── NAV SCROLL ─────────────────────────────────────────
const nav = document.getElementById('nav');
window.addEventListener('scroll', () => {
  nav.classList.toggle('scrolled', window.scrollY > 40);
}, { passive: true });

// ── HAMBURGER ──────────────────────────────────────────
const hamburger = document.getElementById('hamburger');
const mobileMenu = document.getElementById('mobileMenu');
let menuOpen = false;

hamburger.addEventListener('click', () => {
  menuOpen = !menuOpen;
  mobileMenu.classList.toggle('open', menuOpen);
  hamburger.style.opacity = menuOpen ? '0.5' : '1';
  document.body.style.overflow = menuOpen ? 'hidden' : '';
});

document.querySelectorAll('.mobile-link').forEach(link => {
  link.addEventListener('click', () => {
    menuOpen = false;
    mobileMenu.classList.remove('open');
    hamburger.style.opacity = '1';
    document.body.style.overflow = '';
  });
});

// ── SCROLL REVEAL ──────────────────────────────────────
const revealElements = document.querySelectorAll('.reveal');

const observer = new IntersectionObserver((entries) => {
  entries.forEach(entry => {
    if (entry.isIntersecting) {
      entry.target.classList.add('visible');
    }
  });
}, { threshold: 0.12, rootMargin: '0px 0px -40px 0px' });

revealElements.forEach(el => observer.observe(el));

// ── CONTACT FORM ───────────────────────────────────────
const form = document.getElementById('contactForm');
const formSuccess = document.getElementById('formSuccess');

if (form) {
  form.addEventListener('submit', (e) => {
    e.preventDefault();
    const btn = form.querySelector('button[type="submit"]');
    btn.textContent = 'Sending...';
    btn.disabled = true;

    // Simulate submission
    setTimeout(() => {
      form.style.display = 'none';
      formSuccess.style.display = 'block';
    }, 1200);
  });
}

// ── SMOOTH ANCHOR SCROLL ───────────────────────────────
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
  anchor.addEventListener('click', (e) => {
    const target = document.querySelector(anchor.getAttribute('href'));
    if (target) {
      e.preventDefault();
      const offset = 80;
      const top = target.getBoundingClientRect().top + window.scrollY - offset;
      window.scrollTo({ top, behavior: 'smooth' });
    }
  });
});

// ── KPI BAR ANIMATION RESET ────────────────────────────
// Re-trigger CSS animations when KPI cards come into view
const kpiFills = document.querySelectorAll('.kpi-fill, .bar');
const kpiObserver = new IntersectionObserver((entries) => {
  entries.forEach(entry => {
    if (entry.isIntersecting) {
      entry.target.style.animation = 'none';
      // eslint-disable-next-line no-unused-expressions
      entry.target.offsetHeight; // trigger reflow
      entry.target.style.animation = '';
    }
  });
}, { threshold: 0.5 });

kpiFills.forEach(el => kpiObserver.observe(el));
