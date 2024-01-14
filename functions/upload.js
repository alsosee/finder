export async function onRequest(context) {
  try {
    switch (context.request.method) {
      case 'PUT':
        const key = context.request.headers.get("x-file-name");
        await context.env.MEDIA.put(key, context.request.body);
        return new Response(`Put ${key} successfully!`);

      default:
        return new Response(`${context.request.method} is not allowed.`, {
          status: 405,
          headers: {
            Allow: 'PUT',
          },
        });
    }
  } catch (err) {
    return new Response(err.stack || err, { status: 500 });
  }
}
