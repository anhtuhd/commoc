declare namespace Cloudflare {
  interface Env {
    DB?: D1Database;
    OWNER_SETUP_ENABLED?: string;
    BUCKET?: R2Bucket;
  }
}
