export async function onRequest(context) {
    try {
      switch (context.request.method) {
        case 'GET':
            // get "url" GET query parameter
            const url = decodeURIComponent(context.request.url.split('?url=')[1]);
            console.log(url);
            // url decode "url" parameter
            // send request to specified URL
            const resp = await fetch(url) // fetch from origin
                .then(response => {
                    console.log(response);
                    return response.blob()
                }) // get response as blob
                .then(blob => new Response(blob, { // create response from blob
                    headers: {
                        'Content-Type': 'image/jpeg',
                    }
                }))
            return resp;    // return response
  
        default:
          return new Response(
            JSON.stringify({
              error: {
                message: 'Method Not Allowed',
              }
            }),
            {
              status: 405,
              headers: {
                Allow: 'GET',
              },
            }
          );
      }
    } catch (err) {
      return new Response(
        JSON.stringify({ error: err.stack || err }),
        { status: 500 }
      );
    }
  }
  